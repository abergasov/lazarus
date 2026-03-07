package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"lazarus/internal/config"
)

type OpenAIAdapter struct {
	id     string
	client *openai.Client
}

func NewOpenAIAdapter(cfg config.ProviderConfig) (*OpenAIAdapter, error) {
	ocfg := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		ocfg.BaseURL = cfg.BaseURL
	}
	return &OpenAIAdapter{id: cfg.ID, client: openai.NewClientWithConfig(ocfg)}, nil
}

func (a *OpenAIAdapter) ID() string { return a.id }

func (a *OpenAIAdapter) Stream(ctx context.Context, req *Request) (<-chan Event, error) {
	ch := make(chan Event, 32)

	msgs := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.System,
		})
	}
	for _, m := range req.Messages {
		// OpenAI requires one message per tool result, so split tool messages
		if m.Role == "tool" && len(m.ToolResults) > 1 {
			for _, tr := range m.ToolResults {
				msgs = append(msgs, openai.ChatCompletionMessage{
					Role:       "tool",
					ToolCallID: tr.CallID,
					Content:    string(tr.Result),
				})
			}
		} else {
			msgs = append(msgs, convertMessage(m))
		}
	}

	// If images are present, attach them to the last user message as multipart content
	if len(req.Images) > 0 {
		lastIdx := -1
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == openai.ChatMessageRoleUser {
				lastIdx = i
				break
			}
		}
		if lastIdx >= 0 {
			parts := []openai.ChatMessagePart{
				{Type: openai.ChatMessagePartTypeText, Text: msgs[lastIdx].Content},
			}
			for _, img := range req.Images {
				mime := img.MimeType
				if mime == "" {
					mime = "image/png"
				}
				dataURL := fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(img.Data))
				parts = append(parts, openai.ChatMessagePart{
					Type:     openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{URL: dataURL, Detail: openai.ImageURLDetailAuto},
				})
			}
			msgs[lastIdx].Content = ""
			msgs[lastIdx].MultiContent = parts
		}
	}

	// Convert tool definitions
	var tools []openai.Tool
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

	streamReq := openai.ChatCompletionRequest{
		Model:     req.Model,
		Messages:  msgs,
		MaxTokens: req.MaxTokens,
		Stream:    true,
		Tools:     tools,
	}

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				slog.Error("openai stream panic", "error", r, "stack", string(debug.Stack()))
				ch <- Event{Type: EventTypeError, Error: fmt.Errorf("internal error in LLM stream")}
			}
		}()

		// 3-minute timeout for the full stream — prevents indefinite hangs
		streamCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()

		stream, err := a.client.CreateChatCompletionStream(streamCtx, streamReq)
		if err != nil {
			ch <- Event{Type: EventTypeError, Error: fmt.Errorf("openai stream: %w", err)}
			return
		}
		defer stream.Close()

		// Accumulate tool calls across stream chunks (OpenAI sends them in pieces)
		toolCalls := map[int]*ToolCall{}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				// Emit accumulated tool calls
				for _, tc := range toolCalls {
					ch <- Event{Type: EventTypeToolCall, ToolCall: tc}
				}
				ch <- Event{Type: EventTypeDone}
				return
			}
			if err != nil {
				ch <- Event{Type: EventTypeError, Error: err}
				return
			}
			if len(resp.Choices) == 0 {
				continue
			}
			delta := resp.Choices[0].Delta

			if delta.Content != "" {
				ch <- Event{Type: EventTypeText, Text: delta.Content}
			}

			// Accumulate streamed tool calls
			for _, tc := range delta.ToolCalls {
				idx := 0
				if tc.Index != nil {
					idx = *tc.Index
				}
				existing, ok := toolCalls[idx]
				if !ok {
					existing = &ToolCall{ID: tc.ID, Name: tc.Function.Name}
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
	}()

	return ch, nil
}

// convertMessage converts our Message to an OpenAI ChatCompletionMessage,
// handling tool calls and tool results.
func convertMessage(m Message) openai.ChatCompletionMessage {
	msg := openai.ChatCompletionMessage{
		Role:    m.Role,
		Content: m.Content,
	}

	// Assistant messages with tool calls
	if m.Role == "assistant" && len(m.ToolCalls) > 0 {
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

func (a *OpenAIAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
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
	result := make([]float32, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		result[i] = float32(v)
	}
	return result, nil
}
