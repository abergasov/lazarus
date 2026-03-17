package adapter_openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/utils"
	"time"

	"github.com/sashabaranov/go-openai"
)

type Service struct {
	ctx context.Context
	log logger.AppLogger

	client *openai.Client
}

func NewService(ctx context.Context, log logger.AppLogger, cfg *entities.AIProvider) *Service {
	return &Service{
		ctx:    ctx,
		client: openai.NewClientWithConfig(openai.DefaultConfig(cfg.APIKey)),
		log:    log.With(logger.WithService("adapter_openai")),
	}
}

func (a *Service) ID() string {
	return "openai"
}

func (a *Service) Embed(ctx context.Context, text string) ([]float32, error) {
	embedCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := a.client.CreateEmbeddings(embedCtx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.SmallEmbedding3,
	})
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}
	if len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("openai embed: empty response")
	}
	return resp.Data[0].Embedding, nil
}

func (a *Service) Stream(ctx context.Context, req *entities.AgentRequest) (<-chan *entities.AgentEvent, error) {
	ch := make(chan *entities.AgentEvent, 32)
	openAIMessages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)
	if req.SystemPrompt != "" {
		openAIMessages = append(openAIMessages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}
	for _, m := range req.Messages {
		// OpenAI requires one message per tool result, so split tool messages
		if m.Role == "tool" {
			for _, tr := range m.ToolResults {
				openAIMessages = append(openAIMessages, openai.ChatCompletionMessage{
					Role:       "tool",
					ToolCallID: tr.CallID,
					Content:    string(tr.Result),
				})
			}
		} else {
			openAIMessages = append(openAIMessages, convertMessage(m))
		}
	}
	if len(req.Images) > 0 {
		// TODO refactor this. How User is populated? it should be -1 always. is it dead code?
		lastIdx := -1
		for i := len(openAIMessages) - 1; i >= 0; i-- {
			if openAIMessages[i].Role == openai.ChatMessageRoleUser {
				lastIdx = i
				break
			}
		}
		if lastIdx >= 0 {
			parts := make([]openai.ChatMessagePart, 0, len(req.Images)+1)
			parts = append(parts, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: openAIMessages[lastIdx].Content,
			})
			for _, img := range req.Images {
				mime := utils.GetFirstValidString(img.MimeType, "image/png")
				dataURL := fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(img.Data))
				parts = append(parts, openai.ChatMessagePart{
					Type:     openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{URL: dataURL, Detail: openai.ImageURLDetailAuto},
				})
			}
			openAIMessages[lastIdx].Content = ""
			openAIMessages[lastIdx].MultiContent = parts
		}
	}

	// Convert tool definitions
	tools := make([]openai.Tool, 0, len(req.Tools))
	for _, t := range req.Tools {
		var params any
		if len(t.Schema) > 0 {
			_ = json.Unmarshal(t.Schema, &params)
		}
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	go a.processRequestToLLM(ctx, &openai.ChatCompletionRequest{
		Model:               req.Model,
		Messages:            openAIMessages,
		MaxCompletionTokens: req.MaxTokens,
		Stream:              true,
		Tools:               tools,
	}, ch)
	return ch, nil
}

// todo review and debug it
func (a *Service) processRequestToLLM(ctx context.Context, streamReq *openai.ChatCompletionRequest, evtChan chan *entities.AgentEvent) {
	defer close(evtChan)
	// 3-minute timeout for the full stream — prevents indefinite hangs
	streamCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	stream, err := a.client.CreateChatCompletionStream(streamCtx, *streamReq)
	if err != nil {
		evtChan <- &entities.AgentEvent{Type: entities.EventTypeError, Error: fmt.Errorf("openai stream: %w", err)}
		return
	}
	defer stream.Close() //nolint:errcheck // it ok

	// Accumulate tool calls across stream chunks (OpenAI sends them in pieces)
	toolCalls := make(map[int]*entities.AgentToolCall)

	for {
		resp, errR := stream.Recv()
		if errR == io.EOF { //nolint:errorlint // not an error, just end of stream
			// Emit accumulated tool calls
			for _, tc := range toolCalls {
				evtChan <- &entities.AgentEvent{Type: entities.EventTypeToolCall, ToolCall: tc}
			}
			evtChan <- &entities.AgentEvent{Type: entities.EventTypeDone}
			return
		}
		if errR != nil {
			evtChan <- &entities.AgentEvent{Type: entities.EventTypeError, Error: errR}
			return
		}
		if len(resp.Choices) == 0 {
			continue
		}
		delta := resp.Choices[0].Delta
		if delta.Content != "" {
			evtChan <- &entities.AgentEvent{Type: entities.EventTypeText, Text: delta.Content}
		}
		// Accumulate streamed tool calls
		for _, tc := range delta.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			existing, ok := toolCalls[idx]
			if !ok {
				existing = &entities.AgentToolCall{ID: tc.ID, Name: tc.Function.Name}
				toolCalls[idx] = existing
			}
			if tc.ID != "" {
				existing.ID = tc.ID
			}
			if tc.Function.Name != "" {
				existing.Name = tc.Function.Name
			}
			existing.Args = append(existing.Args, []byte(tc.Function.Arguments)...)
		}
	}
}

// convertMessage converts our Message to an OpenAI ChatCompletionMessage,
// handling tool calls and tool results.
// TODO this is messy, debug and refactor it
func convertMessage(m *entities.AgentRequestMessage) openai.ChatCompletionMessage {
	msg := openai.ChatCompletionMessage{
		Role:    m.Role.String(),
		Content: m.Content,
	}

	// Assistant messages with tool calls
	if m.Role == entities.RoleAIModel && len(m.ToolCalls) > 0 {
		for _, tc := range m.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, openai.ToolCall{
				ID:   tc.ID,
				Type: openai.ToolTypeFunction,
				Function: openai.FunctionCall{
					Name:      tc.Name,
					Arguments: string(tc.Args),
				},
			})
		}
	}

	// Tool result messages: OpenAI expects one message per tool result with role=tool
	if m.Role == "tool" && len(m.ToolResults) > 0 {
		// Return first result; caller should split into multiple messages
		msg.ToolCallID = m.ToolResults[0].CallID
		msg.Content = string(m.ToolResults[0].Result)
	}

	return msg
}
