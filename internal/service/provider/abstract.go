package provider

import (
	"context"
	"lazarus/internal/entities"
)

type Provider interface {
	ID() string
	Stream(ctx context.Context, req *entities.AgentRequest) (<-chan *entities.AgentEvent, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}
