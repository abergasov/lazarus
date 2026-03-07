package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"lazarus/internal/config"
)

type AnthropicAdapter struct {
	id     string
	client *anthropic.Client
}

func NewAnthropicAdapter(cfg config.ProviderConfig) (*AnthropicAdapter, error) {
	client := anthropic.NewClient(option.WithAPIKey(cfg.APIKey))
	return &AnthropicAdapter{id: cfg.ID, client: &client}, nil
}

func (a *AnthropicAdapter) ID() string { return a.id }

func (a *AnthropicAdapter) Stream(ctx context.Context, req *Request) (<-chan Event, error) {
	ch := make(chan Event, 32)

	msgs := make([]anthropic.MessageParam, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch m.Role {
		case "user":
			content := []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(m.Content),
			}
			for _, img := range req.Images {
				b64 := base64.StdEncoding.EncodeToString(img.Data)
				content = append([]anthropic.ContentBlockParamUnion{
					anthropic.NewImageBlockBase64(img.MimeType, b64),
				}, content...)
			}
			msgs = append(msgs, anthropic.NewUserMessage(content...))
		case "assistant":
			blocks := []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(m.Content),
			}
			for _, tc := range m.ToolCalls {
				var input any
				_ = json.Unmarshal(tc.Args, &input)
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
			}
			msgs = append(msgs, anthropic.NewAssistantMessage(blocks...))
		case "tool":
			var content []anthropic.ContentBlockParamUnion
			for _, tr := range m.ToolResults {
				isErr := tr.IsError
				_ = isErr
				content = append(content, anthropic.NewToolResultBlock(tr.CallID, string(tr.Result), tr.IsError))
			}
			msgs = append(msgs, anthropic.NewUserMessage(content...))
		}
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: int64(req.MaxTokens),
		Messages:  msgs,
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: req.System},
		}
	}
	if len(req.Tools) > 0 {
		tools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
		for _, t := range req.Tools {
			var props any
			_ = json.Unmarshal(t.Schema, &props)
			tools = append(tools, anthropic.ToolUnionParamOfTool(
				anthropic.ToolInputSchemaParam{Properties: props},
				t.Name,
			))
			// Set description via the OfTool field
			if tools[len(tools)-1].OfTool != nil {
				tools[len(tools)-1].OfTool.Description = anthropic.String(t.Description)
			}
		}
		params.Tools = tools
	}

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				slog.Error("anthropic stream panic", "error", r, "stack", string(debug.Stack()))
				ch <- Event{Type: EventTypeError, Error: fmt.Errorf("internal error in LLM stream")}
			}
		}()

		stream := a.client.Messages.NewStreaming(ctx, params)

		// track pending tool call being built
		var (
			pendingToolID   string
			pendingToolName string
			pendingToolArgs strings.Builder
		)

		for stream.Next() {
			ev := stream.Current()

			// Use the Type field to switch
			switch ev.Type {
			case "content_block_start":
				block := ev.AsContentBlockStart()
				if block.ContentBlock.Type == "tool_use" {
					tu := block.ContentBlock.AsToolUse()
					pendingToolID = tu.ID
					pendingToolName = tu.Name
					pendingToolArgs.Reset()
				}
			case "content_block_delta":
				block := ev.AsContentBlockDelta()
				switch block.Delta.Type {
				case "text_delta":
					ch <- Event{Type: EventTypeText, Text: block.Delta.AsTextDelta().Text}
				case "input_json_delta":
					pendingToolArgs.WriteString(block.Delta.AsInputJSONDelta().PartialJSON)
				}
			case "content_block_stop":
				if pendingToolID != "" {
					ch <- Event{
						Type: EventTypeToolCall,
						ToolCall: &ToolCall{
							ID:   pendingToolID,
							Name: pendingToolName,
							Args: []byte(pendingToolArgs.String()),
						},
					}
					pendingToolID = ""
					pendingToolName = ""
					pendingToolArgs.Reset()
				}
			case "message_start":
				start := ev.AsMessageStart()
				ch <- Event{
					Type:        EventTypeUsage,
					InputTokens: int(start.Message.Usage.InputTokens),
				}
			case "message_delta":
				delta := ev.AsMessageDelta()
				ch <- Event{
					Type:         EventTypeUsage,
					OutputTokens: int(delta.Usage.OutputTokens),
				}
			}
		}
		if err := stream.Err(); err != nil {
			ch <- Event{Type: EventTypeError, Error: fmt.Errorf("anthropic stream: %w", err)}
			return
		}
		ch <- Event{Type: EventTypeDone}
	}()

	return ch, nil
}

func (a *AnthropicAdapter) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, fmt.Errorf("anthropic adapter does not support embeddings: use openai embed role")
}
