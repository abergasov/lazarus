package entities

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type AgentProvider string

const (
	AgentProviderAnthropic    AgentProvider = "anthropic"
	AgentProviderOpenAI       AgentProvider = "openai"
	AgentProviderOpenAICompat AgentProvider = "openai_compat"
	AgentProviderOpenRouter   AgentProvider = "openrouter"
)

func (ap *AgentProvider) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("agent provider must be string")
	}

	v := AgentProvider(value.Value)

	if !v.IsValid() {
		return fmt.Errorf("invalid agent provider: %s", value.Value)
	}

	*ap = v
	return nil
}

func (ap *AgentProvider) IsValid() bool {
	validMap := map[string]struct{}{
		string(AgentProviderAnthropic):    {},
		string(AgentProviderOpenAI):       {},
		string(AgentProviderOpenAICompat): {},
		string(AgentProviderOpenRouter):   {},
	}

	_, ok := validMap[string(*ap)]
	return ok
}

type AIProvider struct {
	ID           string        `yaml:"id"`
	Type         AgentProvider `yaml:"type"` // "anthropic" | "openai" | "openai_compat" | "openrouter"
	APIKey       string        `yaml:"api_key"`
	BaseURL      string        `yaml:"base_url,omitempty"`
	DefaultModel string        `yaml:"default_model"`
}
