package provider

import (
	"context"
	"fmt"
	"io"

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
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	streamReq := openai.ChatCompletionRequest{
		Model:     req.Model,
		Messages:  msgs,
		MaxTokens: req.MaxTokens,
		Stream:    true,
	}

	go func() {
		defer close(ch)

		stream, err := a.client.CreateChatCompletionStream(ctx, streamReq)
		if err != nil {
			ch <- Event{Type: EventTypeError, Error: fmt.Errorf("openai stream: %w", err)}
			return
		}
		defer stream.Close()

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				ch <- Event{Type: EventTypeDone}
				return
			}
			if err != nil {
				ch <- Event{Type: EventTypeError, Error: err}
				return
			}
			if len(resp.Choices) > 0 && resp.Choices[0].Delta.Content != "" {
				ch <- Event{Type: EventTypeText, Text: resp.Choices[0].Delta.Content}
			}
		}
	}()

	return ch, nil
}

func (a *OpenAIAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := a.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.SmallEmbedding3,
	})
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}
	result := make([]float32, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		result[i] = float32(v)
	}
	return result, nil
}
