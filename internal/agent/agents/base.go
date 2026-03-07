package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"lazarus/internal/agent/tools"
	"lazarus/internal/entities"
	"lazarus/internal/provider"

	"github.com/jmoiron/sqlx"
)

const maxIterations = 8

type agentBase struct {
	prov     provider.Provider
	model    string
	registry *tools.Registry
	auditDB  *sqlx.DB
}

type toolExecResult struct {
	CallID     string
	Name       string
	Result     any
	Error      error
	DurationMs int
}

func (a *agentBase) runLoop(
	ctx context.Context,
	session interface{ GetMessages() []provider.Message; AppendMessage(provider.Message) },
	assembledCtx interface{ GetSystemPromptContext() string },
	phase string,
	systemPrompt string,
	userMsg string,
	userID string,
	visitID string,
	out chan<- entities.ClientEvent,
) error {
	availableTools := a.registry.ForPhase(phase)
	toolDefs := make([]provider.ToolDef, len(availableTools))
	for i, t := range availableTools {
		toolDefs[i] = provider.ToolDef{
			Name:        t.Name,
			Description: t.Description,
			Schema:      t.Schema,
		}
	}

	messages := append(session.GetMessages(), provider.Message{Role: "user", Content: userMsg})

	// Inject today's date into all system prompts
	resolvedPrompt := strings.ReplaceAll(systemPrompt, "{{TODAY}}", time.Now().Format("2006-01-02"))

	uc := &tools.UserContext{Phase: phase, UserID: userID, VisitID: visitID}

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Warn the LLM when approaching iteration limit
		iterationSystem := resolvedPrompt + "\n\n" + assembledCtx.GetSystemPromptContext()
		if iteration >= maxIterations-2 {
			iterationSystem += fmt.Sprintf(
				"\n\n⚠️ ITERATION WARNING: You are on iteration %d of %d. Wrap up your analysis and provide a final response. Do NOT call more tools unless absolutely critical for patient safety.",
				iteration+1, maxIterations)
		}

		req := &provider.Request{
			Model:     a.model,
			System:    iterationSystem,
			Messages:  messages,
			Tools:     toolDefs,
			MaxTokens: 4096,
		}

		eventCh, err := a.prov.Stream(ctx, req)
		if err != nil {
			return err
		}

		var pendingToolCalls []provider.ToolCall
		var assistantText string

		for ev := range eventCh {
			switch ev.Type {
			case provider.EventTypeText:
				assistantText += ev.Text
				out <- entities.ClientEvent{
					Type:    entities.EventTextDelta,
					Payload: entities.TextDeltaPayload{Text: ev.Text},
				}
			case provider.EventTypeToolCall:
				pendingToolCalls = append(pendingToolCalls, *ev.ToolCall)
				out <- entities.ClientEvent{
					Type: entities.EventToolStart,
					Payload: entities.ToolStartPayload{
						StepID: ev.ToolCall.ID,
						Tool:   ev.ToolCall.Name,
						Label:  a.registry.HumanLabel(ev.ToolCall.Name, ev.ToolCall.Args),
					},
				}
			case provider.EventTypeError:
				return ev.Error
			}
		}

		// Apply output safety filter before sending to user
		if assistantText != "" {
			if filtered := filterUnsafeOutput(assistantText); filtered != assistantText {
				slog.Warn("output filter modified response",
					"phase", phase,
					"user_id", userID,
					"iteration", iteration,
					"original_len", len(assistantText),
					"filtered_len", len(filtered),
				)
				assistantText = filtered
			}
		}

		if len(pendingToolCalls) == 0 {
			break
		}

		// Execute tools concurrently
		results := make([]toolExecResult, len(pendingToolCalls))
		var wg sync.WaitGroup
		for i, tc := range pendingToolCalls {
			wg.Add(1)
			go func(idx int, call provider.ToolCall) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						slog.Error("tool execution panic", "tool", call.Name, "error", r, "stack", string(debug.Stack()))
						results[idx] = toolExecResult{CallID: call.ID, Name: call.Name, Error: fmt.Errorf("tool %s panicked: %v", call.Name, r)}
					}
				}()
				start := time.Now()
				result, err := a.registry.Execute(ctx, call.Name, call.Args, uc)
				durationMs := int(time.Since(start).Milliseconds())
				results[idx] = toolExecResult{CallID: call.ID, Name: call.Name, Result: result, Error: err, DurationMs: durationMs}
				success := err == nil
				summary := a.registry.Summary(call.Name, result)
				out <- entities.ClientEvent{
					Type: entities.EventToolDone,
					Payload: entities.ToolDonePayload{
						StepID:  call.ID,
						Success: success,
						Summary: summary,
					},
				}
			}(i, tc)
		}
		wg.Wait()

		// Audit log all tool calls
		a.logToolCalls(ctx, userID, visitID, phase, iteration, pendingToolCalls, results)

		// Append assistant turn + tool results
		assistantMsg := provider.Message{Role: "assistant", Content: assistantText}
		for _, tc := range pendingToolCalls {
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, tc)
		}
		messages = append(messages, assistantMsg)

		toolResultMsg := provider.Message{Role: "tool"}
		for _, r := range results {
			rb, _ := json.Marshal(r.Result)
			if r.Error != nil {
				rb, _ = json.Marshal(map[string]string{"error": r.Error.Error()})
			}
			toolResultMsg.ToolResults = append(toolResultMsg.ToolResults, provider.ToolResult{
				CallID:  r.CallID,
				Result:  rb,
				IsError: r.Error != nil,
			})
		}
		messages = append(messages, toolResultMsg)
	}

	return nil
}

// logToolCalls persists tool execution details for audit trail
func (a *agentBase) logToolCalls(ctx context.Context, userID, visitID, phase string, iteration int, calls []provider.ToolCall, results []toolExecResult) {
	if a.auditDB == nil {
		return
	}
	for i, call := range calls {
		result := results[i]
		resultJSON, _ := json.Marshal(result.Result)
		if result.Error != nil {
			resultJSON, _ = json.Marshal(map[string]string{"error": result.Error.Error()})
		}

		var visitIDPtr *string
		if visitID != "" {
			visitIDPtr = &visitID
		}

		_, err := a.auditDB.ExecContext(ctx, `
			INSERT INTO agent_audit_log (user_id, visit_id, phase, iteration, tool_name, tool_args, tool_result, is_error, duration_ms)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, userID, visitIDPtr, phase, iteration, call.Name, call.Args, resultJSON, result.Error != nil, result.DurationMs)
		if err != nil {
			slog.Error("failed to write audit log", "error", err, "tool", call.Name)
		}
	}
}

// sessionAdapter wraps Session to satisfy the runLoop interface
type sessionAdapter struct {
	sess *sessionRef
}

type sessionRef struct {
	messages []provider.Message
}

func (s *sessionRef) GetMessages() []provider.Message { return s.messages }
func (s *sessionRef) AppendMessage(m provider.Message) { s.messages = append(s.messages, m) }

// contextAdapter wraps AssembledContext
type contextAdapter struct {
	promptCtx string
}

func (c *contextAdapter) GetSystemPromptContext() string { return c.promptCtx }
