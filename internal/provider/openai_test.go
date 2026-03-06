package provider_test

import (
	"context"
	"os"
	"testing"

	"lazarus/internal/config"
	"lazarus/internal/provider"

	"github.com/stretchr/testify/require"
)

func TestOpenAIAdapter_Stream(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	cfg := config.ProviderConfig{
		ID:     "openai",
		Type:   "openai",
		APIKey: key,
	}
	adapter, err := provider.NewOpenAIAdapter(cfg)
	require.NoError(t, err)

	req := &provider.Request{
		Model:     "gpt-4o-mini",
		System:    "Be brief.",
		Messages:  []provider.Message{{Role: "user", Content: "Say: hello"}},
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

func TestOpenAIAdapter_Embed(t *testing.T) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	cfg := config.ProviderConfig{ID: "openai", Type: "openai", APIKey: key}
	adapter, err := provider.NewOpenAIAdapter(cfg)
	require.NoError(t, err)

	vec, err := adapter.Embed(context.Background(), "glucose lab test blood sugar")
	require.NoError(t, err)
	require.Len(t, vec, 1536)
}
