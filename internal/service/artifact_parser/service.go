package artifact_parser

import (
	"context"
	"lazarus/internal/config"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"lazarus/internal/storage/bucket"
)

type Service struct {
	ctx          context.Context
	cfg          *config.AppConfig
	log          logger.AppLogger
	repo         *repository.Repo
	bucketClient *bucket.Client
}

func NewService(
	ctx context.Context,
	log logger.AppLogger,
	cfg *config.AppConfig,
	repo *repository.Repo,
	bucketClient *bucket.Client,
) *Service {
	return &Service{
		ctx:          ctx,
		cfg:          cfg,
		log:          log.With(logger.WithService("artifact_parser")),
		repo:         repo,
		bucketClient: bucketClient,
	}
}

func (s *Service) Run() {
	//
}

func (s *Service) Stop() {}
