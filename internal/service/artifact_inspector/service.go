package artifact_inspector

import (
	"context"
	"lazarus/internal/config"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
)

// Service get all uploaded artifacts from the database
// detect their mime type and content summary, and update the database with the results
// all runs in isolated container in case of some malicious file that can harm the system
type Service struct {
	ctx  context.Context
	cfg  *config.AppConfig
	log  logger.AppLogger
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

func (s *Service) Run() {
	for {

	}
}

func (s *Service) Stop() {

}
