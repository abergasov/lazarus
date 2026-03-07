package artifact_manager

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"lazarus/internal/storage/bucket"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"unicode"

	"github.com/google/uuid"
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
	if cfg.MaxUploadSizeBytes <= 0 {
		log.Fatal("invalid MaxUploadSizeBytes configuration", fmt.Errorf("max upload size must be positive (got %d)", cfg.MaxUploadSizeBytes))
	}
	if err := os.MkdirAll(cfg.RawUploadsDir, 0o700); err != nil {
		log.Fatal("cannot create raw uploads dir", err, logger.WithPath(cfg.RawUploadsDir))
	}
	return &Service{
		ctx:          ctx,
		cfg:          cfg,
		log:          log,
		repo:         repo,
		bucketClient: bucketClient,
	}
}

func (s *Service) Upload(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) (*entities.Artifact, error) {
	if file == nil {
		return nil, errors.New("file is nil")
	}
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open upload: %w", err)
	}
	defer src.Close() //nolint:errcheck // we will remove the file in defer, so no need to check close error

	tmp, err := os.CreateTemp(s.cfg.RawUploadsDir, "upload-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	limited := &io.LimitedReader{R: src, N: s.cfg.MaxUploadSizeBytes + 1}
	hasher := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, hasher), limited)
	if err != nil {
		return nil, fmt.Errorf("copy upload: %w", err)
	}
	if n == 0 {
		return nil, errors.New("empty file")
	}
	if n > s.cfg.MaxUploadSizeBytes {
		return nil, errors.New("file too large")
	}
	if _, err = tmp.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind temp file for mime detection: %w", err)
	}
	head := make([]byte, 512)
	headN, err := io.ReadFull(tmp, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read head: %w", err)
	}
	detectedMIME := http.DetectContentType(head[:headN])
	if _, err = tmp.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind temp file for upload: %w", err)
	}
	artifactID := uuid.New()
	objectKey := fmt.Sprintf("%s/%s", userID.String(), artifactID.String())
	if err = s.bucketClient.Upload(ctx, objectKey, tmp, n); err != nil {
		return nil, fmt.Errorf("upload to bucket: %w", err)
	}

	artifact := &entities.Artifact{
		ID:             artifactID,
		OwnerID:        userID,
		Kind:           entities.ArtifactKindOther,
		Status:         entities.ArtifactStatusQuarantined,
		DeclaredMIME:   file.Header.Get("Content-Type"),
		DetectedMIME:   detectedMIME,
		OriginalName:   SafeName(file.Filename),
		ByteSize:       n,
		SHA256Hex:      hex.EncodeToString(hasher.Sum(nil)),
		Storage:        entities.ArtifactStorageS3,
		Bucket:         s.cfg.S3.Bucket,
		ObjectKey:      objectKey,
		ContentSummary: "",
		MetaJSON:       sql.Null[json.RawMessage]{},
	}
	if err = s.repo.CreateArtifact(ctx, artifactID, artifact); err != nil {
		_ = s.bucketClient.Delete(ctx, objectKey)
		return nil, fmt.Errorf("create artifact record: %w", err)
	}
	return artifact, nil
}

func (s *Service) GetArtifactByID(ctx context.Context, artifactID, userID uuid.UUID) (*entities.Artifact, error) {
	return s.repo.GetArtifactByID(ctx, userID, artifactID)
}

func (s *Service) ListArtifactsByUser(ctx context.Context, userID uuid.UUID) ([]*entities.Artifact, error) {
	return s.repo.GetAllArtifactsByOwner(ctx, userID)
}

func (s *Service) DeleteArtifact(ctx context.Context, artifactID, userID uuid.UUID) error {
	return s.repo.DeleteArtifact(ctx, userID, artifactID)
}

func SafeName(s string) string {
	var b strings.Builder
	maxFileName := 120
	b.Grow(min(len([]rune(s)), maxFileName))

	lastWasSpace := false
	for _, r := range []rune(s) { //nolint:staticcheck // check each rune
		if len([]rune(b.String())) >= maxFileName {
			break
		}
		if r < 32 || r == 127 {
			continue
		}
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			r = '_'
		}
		if unicode.IsSpace(r) {
			if b.Len() == 0 || lastWasSpace {
				continue
			}
			r = ' '
			lastWasSpace = true
		} else {
			lastWasSpace = false
		}
		b.WriteRune(r)
	}
	out := strings.ReplaceAll(b.String(), ". ", "")
	out = strings.TrimSpace(out)
	out = strings.Trim(out, ".")
	if out == "" {
		return "file"
	}
	switch strings.ToUpper(out) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return "file_" + out
	}

	return out
}
