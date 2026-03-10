package adapter_anthropic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Service struct {
	ctx context.Context
	log logger.AppLogger

	client *anthropic.Client
}

func NewService(ctx context.Context, log logger.AppLogger, cfg *entities.AIProvider) *Service {
	return &Service{
		ctx:    ctx,
		client: new(anthropic.NewClient(option.WithAPIKey(cfg.APIKey))),
		log:    log.With(logger.WithService("adapter_anthropic")),
	}
}

func (a *Service) ID() string {
	return "anthropic"
}

func (a *Service) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, fmt.Errorf("anthropic adapter does not support embeddings: use openai embed role")
}

func (a *Service) Stream(ctx context.Context, req *entities.AgentRequest) (<-chan *entities.AgentEvent, error) {
	ch := make(chan *entities.AgentEvent, 32)

	antropicMsg := make([]anthropic.MessageParam, 0, len(req.Messages))
	for i := range req.Messages {
		switch req.Messages[i].Role {
		case entities.RoleUser:
			content := []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(req.Messages[i].Content),
			}
			for _, img := range req.Images {
				b64 := base64.StdEncoding.EncodeToString(img.Data)
				content = append([]anthropic.ContentBlockParamUnion{
					anthropic.NewImageBlockBase64(img.MimeType, b64),
				}, content...)
			}
			antropicMsg = append(antropicMsg, anthropic.NewUserMessage(content...))
		case entities.RoleAIModel:
			blocks := []anthropic.ContentBlockParamUnion{
				anthropic.NewTextBlock(req.Messages[i].Content),
			}
			for _, tc := range req.Messages[i].ToolCalls {
				var input any
				if err := json.Unmarshal(tc.Args, &input); err != nil {
					return nil, fmt.Errorf("failed to unmarshal ai model input: %w", err)
				}
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
			}
			antropicMsg = append(antropicMsg, anthropic.NewAssistantMessage(blocks...))
		case entities.RoleTool:
			var content []anthropic.ContentBlockParamUnion
			for _, tr := range req.Messages[i].ToolResults {
				content = append(content, anthropic.NewToolResultBlock(tr.CallID, string(tr.Result), tr.IsError))
			}
			antropicMsg = append(antropicMsg, anthropic.NewUserMessage(content...))
		}
	}
	params := anthropic.MessageNewParams{
		Model:     req.Model,
		MaxTokens: int64(req.MaxTokens),
		Messages:  antropicMsg,
	}
	if req.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{
				Text: req.SystemPrompt,
			},
		}
	}
	if len(req.Tools) > 0 {
		params.Tools = make([]anthropic.ToolUnionParam, 0, len(req.Tools))
		for _, t := range req.Tools {
			var props any
			if err := json.Unmarshal(t.Schema, &props); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tools input: %w", err)
			}
			params.Tools = append(params.Tools, anthropic.ToolUnionParamOfTool(
				anthropic.ToolInputSchemaParam{Properties: props},
				t.Name,
			))
			// Set description via the OfTool field
			if params.Tools[len(params.Tools)-1].OfTool != nil {
				params.Tools[len(params.Tools)-1].OfTool.Description = anthropic.String(t.Description)
			}
		}
	}
	go a.processRequestToLLM(ctx, &params, ch)
	return ch, nil
}

func (a *Service) processRequestToLLM(ctx context.Context, params *anthropic.MessageNewParams, evtChan chan *entities.AgentEvent) {
	stream := a.client.Messages.NewStreaming(ctx, *params)
	defer close(evtChan)

	// track pending tool call being built
	var (
		pendingToolID   string
		pendingToolName string
		pendingToolArgs strings.Builder
	)

	for stream.Next() {
		ev := stream.Current()
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
				evtChan <- &entities.AgentEvent{
					Type: entities.EventTypeText,
					Text: block.Delta.AsTextDelta().Text,
				}
			case "input_json_delta":
				pendingToolArgs.WriteString(block.Delta.AsInputJSONDelta().PartialJSON)
			}
		case "content_block_stop":
			if pendingToolID != "" {
				evtChan <- &entities.AgentEvent{
					Type: entities.EventTypeToolCall,
					ToolCall: &entities.AgentToolCall{
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
			evtChan <- &entities.AgentEvent{
				Type:        entities.EventTypeUsage,
				InputTokens: start.Message.Usage.InputTokens,
			}
		case "message_delta":
			delta := ev.AsMessageDelta()
			evtChan <- &entities.AgentEvent{
				Type:         entities.EventTypeUsage,
				OutputTokens: delta.Usage.OutputTokens,
			}
		}
		if err := stream.Err(); err != nil {
			evtChan <- &entities.AgentEvent{
				Type:  entities.EventTypeError,
				Error: fmt.Errorf("anthropic stream: %w", err),
			}
			return
		}
	}
	evtChan <- &entities.AgentEvent{Type: entities.EventTypeDone}
}
