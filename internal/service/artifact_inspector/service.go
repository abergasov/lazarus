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
	"net/http"
	"os"
	"time"
)

const (
	sniffLen = 512
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
}

func NewService(
	ctx context.Context,
	log logger.AppLogger,
	cfg *config.AppConfig,
	repo *repository.Repo,
	bucketClient *bucket.Client,
	avClient *antivirus.Client,
) *Service {
	return &Service{
		ctx:          ctx,
		cfg:          cfg,
		log:          log,
		repo:         repo,
		bucketClient: bucketClient,
		avClient:     avClient,
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
	defer os.Remove(tmp.Name()) //nolint:errcheck
	defer tmp.Close()           //nolint:errcheck

	rCloser, err := s.bucketClient.Download(s.ctx, artifact.ObjectKey)
	if err != nil {
		return fmt.Errorf("cannot download artifact: %w", err)
	}
	defer rCloser.Close() //nolint:errcheck

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
	detectedMime, err := s.detectMimeType(tmp)
	if err != nil {
		return fmt.Errorf("cannot detect mime type: %w", err)
	}
	if detectedMime != artifact.DetectedMIME {
		if err = s.purgeArtifact(ctx, artifact); err != nil {
			return fmt.Errorf("cannot purge artifact with mime type mismatch: %w", err)
		}
		return fmt.Errorf("mime type mismatch: %s vs %s", detectedMime, artifact.DetectedMIME)
	}
	// todo parse to images, documents, etc and update artifact type in database
	return s.markArtifactClean(ctx, artifact)
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

func (s *Service) detectMimeType(f *os.File) (string, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind file: %w", err)
	}

	buf := make([]byte, sniffLen)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read sniff bytes: %w", err)
	}

	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind file: %w", err)
	}

	return http.DetectContentType(buf[:n]), nil
}
