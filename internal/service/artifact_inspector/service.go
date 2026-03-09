package artifact_inspector

import (
	"context"
	"fmt"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"lazarus/internal/storage/bucket"
	"lazarus/internal/utils"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// Service get all uploaded artifacts from the database
// detect their mime type and content summary, and update the database with the results
// all runs in isolated container in case of some malicious file that can harm the system
type Service struct {
	ctx          context.Context
	cfg          *config.AppConfig
	log          logger.AppLogger
	repo         *repository.Repo
	bucketClient *bucket.Client

	storagePath string
}

func NewService(ctx context.Context, log logger.AppLogger, cfg *config.AppConfig, repo *repository.Repo, bucketClient *bucket.Client) *Service {
	storagePath := os.TempDir()
	if err := os.MkdirAll(storagePath, os.ModePerm); err != nil {
		log.Fatal("cannot create storage dir", err, logger.WithPath(storagePath))
	}
	return &Service{
		ctx:          ctx,
		cfg:          cfg,
		log:          log,
		repo:         repo,
		bucketClient: bucketClient,

		storagePath: storagePath,
	}
}

func (s *Service) Run() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			s.log.Info("artifact inspector service is stopping")
			return
		case <-ticker.C:
			quarantinedArtifacts, err := s.repo.GetQuarantinedArtifacts(s.ctx)
			if err != nil {
				s.log.Error("cannot get quarantined artifacts from database", err)
				continue
			}
			if len(quarantinedArtifacts) == 0 {
				continue
			}
			utils.Shuffle(quarantinedArtifacts)
			for i := range quarantinedArtifacts {
				if err = s.InspectArtifact(quarantinedArtifacts[i]); err != nil {
					s.log.Error("cannot inspect artifact", err, logger.WithArtifactID(quarantinedArtifacts[i].ID))
					break
				}
			}
		}
	}
}

func (s *Service) Stop() {
}

func (s *Service) InspectArtifact(artifact *entities.Artifact) error {
	tmpDir := filepath.Join(os.TempDir(), uuid.NewString())
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return fmt.Errorf("cannot create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	s.bucketClient.Download()

	return nil
}
