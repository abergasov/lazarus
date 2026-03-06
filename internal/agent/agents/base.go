package agents

import (
	"context"
	"encoding/json"
	"sync"

	"lazarus/internal/agent/tools"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
)

const maxIterations = 10

type agentBase struct {
	prov     provider.Provider
	model    string
	registry *tools.Registry
}

type toolExecResult struct {
	CallID string
	Name   string
	Result any
	Error  error
}

func (a *agentBase) runLoop(
	ctx context.Context,
	session interface{ GetMessages() []provider.Message; AppendMessage(provider.Message) },
	assembledCtx interface{ GetSystemPromptContext() string },
	phase string,
	systemPrompt string,
	userMsg string,
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

	uc := &tools.UserContext{Phase: phase}

	for iteration := 0; iteration < maxIterations; iteration++ {
		req := &provider.Request{
			Model:     a.model,
			System:    systemPrompt + "\n\n" + assembledCtx.GetSystemPromptContext(),
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
				result, err := a.registry.Execute(ctx, call.Name, call.Args, uc)
				results[idx] = toolExecResult{CallID: call.ID, Name: call.Name, Result: result, Error: err}
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
