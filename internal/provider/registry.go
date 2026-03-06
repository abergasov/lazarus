package provider

import (
	"fmt"

	"lazarus/internal/config"
)

type Registry struct {
	providers map[string]Provider
	roles     config.RoleConfig
}

func NewRegistry(cfg *config.LLMConfig) (*Registry, error) {
	r := &Registry{
		providers: make(map[string]Provider),
		roles:     cfg.Roles,
	}
	for _, pc := range cfg.Providers {
		p, err := newProvider(pc)
		if err != nil {
			return nil, fmt.Errorf("provider %s: %w", pc.ID, err)
		}
		r.providers[pc.ID] = p
	}
	return r, nil
}

func (r *Registry) ForRole(role string) (Provider, string, error) {
	var rm config.RoleModel
	switch role {
	case "prep":
		rm = r.roles.Prep
	case "during":
		rm = r.roles.During
	case "after":
		rm = r.roles.After
	case "vision":
		rm = r.roles.Vision
	case "embed":
		rm = r.roles.Embed
	default:
		return nil, "", fmt.Errorf("unknown role: %s", role)
	}
	p, ok := r.providers[rm.ProviderID]
	if !ok {
		return nil, "", fmt.Errorf("provider not found: %s", rm.ProviderID)
	}
	return p, rm.Model, nil
}

func (r *Registry) Get(id string) (Provider, error) {
	p, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", id)
	}
	return p, nil
}

func newProvider(cfg config.ProviderConfig) (Provider, error) {
	switch cfg.Type {
	case "anthropic":
		return NewAnthropicAdapter(cfg)
	case "openai", "openai_compat", "openrouter":
		return NewOpenAIAdapter(cfg)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}
