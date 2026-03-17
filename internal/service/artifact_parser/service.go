package artifact_parser

import (
	"context"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"lazarus/internal/service/provider"
	"lazarus/internal/storage/bucket"
	"lazarus/internal/utils"
	"time"
)

type Service struct {
	ctx          context.Context
	cfg          *config.AppConfig
	log          logger.AppLogger
	repo         *repository.Repo
	bucketClient *bucket.Client

	registry *provider.Registry
}

func NewService(
	ctx context.Context,
	log logger.AppLogger,
	cfg *config.AppConfig,
	repo *repository.Repo,
	bucketClient *bucket.Client,
	registry *provider.Registry,
) *Service {
	return &Service{
		ctx:          ctx,
		cfg:          cfg,
		log:          log.With(logger.WithService("artifact_parser")),
		repo:         repo,
		bucketClient: bucketClient,
		registry:     registry,
	}
}

func (s *Service) Run() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			s.log.Info("artifact parser service is stopping")
			return
		case <-ticker.C:
			cleanedArtifacts, err := s.repo.GetCleanedArtifacts(s.ctx)
			if err != nil {
				s.log.Error("cannot get cleaned artifacts from database", err)
				continue
			}
			if len(cleanedArtifacts) == 0 {
				continue
			}
			utils.Shuffle(cleanedArtifacts)
			for i := range cleanedArtifacts {
				if err = s.ProcessArtifact(cleanedArtifacts[i]); err != nil {
					s.log.Error("cannot scan artifact", err, logger.WithArtifactID(cleanedArtifacts[i].ID))
					break
				}
			}
		}
	}
}

func (s *Service) ProcessArtifact(artifact *entities.Artifact) error {
	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Minute)
	defer cancel()

	if err := s.scanImages(ctx, artifact); err != nil {
		return err
	}
	return s.repo.UpdateArtifactStatus(ctx, artifact.ID, entities.ArtifactStatusParsed)
}

func (s *Service) Stop() {}
