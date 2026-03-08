package artifact_parser

import (
	"context"
	"lazarus/internal/config"
	"lazarus/internal/logger"
)

type Service struct {
	ctx context.Context
	cfg *config.AppConfig
	log logger.AppLogger
}

func NewService(ctx context.Context, log logger.AppLogger, cfg *config.AppConfig) *Service {
	return &Service{
		ctx: ctx,
		cfg: cfg,
		log: log.With(logger.WithService("artifact_parser")),
	}
}
