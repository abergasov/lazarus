package user

import (
	"context"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
)

type Service struct {
	ctx context.Context
	cfg *config.AppConfig
	log logger.AppLogger

	repo *repository.Repo
}

func NewService(ctx context.Context, log logger.AppLogger, cfg *config.AppConfig, repo *repository.Repo) *Service {
	return &Service{
		ctx:  ctx,
		cfg:  cfg,
		log:  log,
		repo: repo,
	}
}

func (r *Service) GetUserByID(ctx context.Context, id int64) (*entities.User, error) {
	return r.repo.GetUserByID(ctx, id)
}
