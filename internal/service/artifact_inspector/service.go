package artifact_inspector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"lazarus/internal/storage/antivirus"
	"lazarus/internal/storage/bucket"
	"lazarus/internal/utils"
	"os"
	"time"
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
	avClient     *antivirus.Client

	storagePath string
}

func NewService(
	ctx context.Context,
	log logger.AppLogger,
	cfg *config.AppConfig,
	repo *repository.Repo,
	bucketClient *bucket.Client,
	avClient *antivirus.Client,
) *Service {
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
		avClient:     avClient,

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
	tmp, err := os.CreateTemp(os.TempDir(), "artifact-*")
	if err != nil {
		return fmt.Errorf("cannot create temporary file: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	rCloser, err := s.bucketClient.Download(s.ctx, artifact.ObjectKey)
	if err != nil {
		return fmt.Errorf("cannot download artifact: %w", err)
	}
	defer rCloser.Close()

	h := sha256.New()
	if _, err = io.Copy(io.MultiWriter(tmp, h), rCloser); err != nil {
		return fmt.Errorf("cannot copy artifact: %w", err)
	}
	if _, err = tmp.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind temp file: %w", err)
	}

	sum := hex.EncodeToString(h.Sum(nil))
	if sum != artifact.SHA256Hex {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", artifact.SHA256Hex, sum)
	}
	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Minute)
	defer cancel()

	if err = s.scanTmpFile(ctx, tmp); err != nil {
		return fmt.Errorf("cannot scan artifact: %w", err)
	}
	return nil
}

func (s *Service) scanTmpFile(ctx context.Context, tmp *os.File) error {
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind temp file: %w", err)
	}

	if err := s.avClient.ScanReader(ctx, tmp); err != nil {
		return fmt.Errorf("av scan: %w", err)
	}
	return nil
}
