package provider_test

import (
	"context"
	"os"
	"testing"

	"lazarus/internal/config"
	"lazarus/internal/provider"

	"github.com/stretchr/testify/require"
)

func TestAnthropicAdapter_Stream(t *testing.T) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	cfg := config.ProviderConfig{
		ID:     "anthropic",
		Type:   "anthropic",
		APIKey: key,
	}

	adapter, err := provider.NewAnthropicAdapter(cfg)
	require.NoError(t, err)

	req := &provider.Request{
		Model:  "claude-haiku-4-5-20251001",
		System: "You are a helpful assistant. Be brief.",
		Messages: []provider.Message{
			{Role: "user", Content: "Say exactly: hello"},
		},
		MaxTokens: 20,
	}

	ch, err := adapter.Stream(context.Background(), req)
	require.NoError(t, err)

	var text string
	for ev := range ch {
		require.NoError(t, ev.Error)
		if ev.Type == provider.EventTypeText {
			text += ev.Text
		}
	}
	require.NotEmpty(t, text)
}
