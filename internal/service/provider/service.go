package provider

import (
	"context"
	"fmt"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"lazarus/internal/service/provider/adapter_anthropic"
	"lazarus/internal/service/provider/adapter_openai"
)

type Registry struct {
	ctx           context.Context
	cfg           *config.AppConfig
	log           logger.AppLogger
	repo          *repository.Repo
	providers     map[string]Provider
	roleProviders map[entities.AgentRole]*entities.AgentRoleConfig
}

func NewRegistry(ctx context.Context, log logger.AppLogger, cfg *config.AppConfig, repo *repository.Repo) (*Registry, error) {
	srv := &Registry{
		ctx:           ctx,
		log:           log.With(logger.WithService("provider_registry")),
		repo:          repo,
		cfg:           cfg,
		providers:     make(map[string]Provider),
		roleProviders: make(map[entities.AgentRole]*entities.AgentRoleConfig),
	}
	for _, provider := range cfg.LLM.Providers {
		switch provider.Type {
		case entities.AgentProviderAnthropic:
			srv.providers[provider.Type.String()] = adapter_anthropic.NewService(ctx, log, provider)
		case entities.AgentProviderOpenAI:
			srv.providers[provider.Type.String()] = adapter_openai.NewService(ctx, log, provider)
		default:
			srv.log.Fatal("unsupported provider type", fmt.Errorf("provider type: %s", provider.Type))
		}
	}
	if cfg.LLM.Roles.Vision == nil {
		return nil, fmt.Errorf("vision role config is nil")
	}
	srv.roleProviders[entities.AgentRoleVision] = cfg.LLM.Roles.Vision
	if cfg.LLM.Roles.AfterVisit == nil {
		return nil, fmt.Errorf("after_visit role config is nil")
	}
	srv.roleProviders[entities.AgentRoleAfterVisit] = cfg.LLM.Roles.AfterVisit
	if cfg.LLM.Roles.DuringVisit == nil {
		return nil, fmt.Errorf("during_visit role config is nil")
	}
	srv.roleProviders[entities.AgentRoleDuringVisit] = cfg.LLM.Roles.DuringVisit
	if cfg.LLM.Roles.PrepVisit == nil {
		return nil, fmt.Errorf("prep_visit role config is nil")
	}
	srv.roleProviders[entities.AgentRolePrepVisit] = cfg.LLM.Roles.PrepVisit
	// todo it not mandatory for now, but maybe in future we will need it, so we can add it later
	srv.roleProviders[entities.AgentRoleEmbed] = cfg.LLM.Roles.Embed
	return srv, nil
}

func (r *Registry) GetForRole(role entities.AgentRole) (p Provider, model string, err error) {
	roleProvider, ok := r.roleProviders[role]
	if !ok {
		return nil, "", fmt.Errorf("provider for role not found: %s", role)
	}
	provider, err := r.GetProvider(roleProvider.ProviderID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get provider for role: %w", err)
	}
	return provider, roleProvider.Model, nil
}

func (r *Registry) GetProvider(name entities.AgentProvider) (Provider, error) {
	provider, ok := r.providers[name.String()]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}
