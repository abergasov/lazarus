package repository

import (
	"context"
	"fmt"
	"lazarus/internal/entities"
	"lazarus/internal/storage/database"
	"lazarus/internal/utils"
	"strings"
	"time"

	"github.com/google/uuid"
)

const TableArtifactDerivatives = "artifact_derivatives"

var (
	artifactDerivativesColumns = []string{
		"d_id",
		"artifact_id",
		"kind",
		"page_num",
		"storage",
		"bucket",
		"object_key",
		"detected_mime_type",
		"byte_size",
		"sha256_hex",
		"created_at",
	}
	artifactDerivativesColumnsStr = strings.Join(artifactDerivativesColumns, ",")
)

func (r *Repo) CreateArtifactDerivative(ctx context.Context, a *entities.ArtifactDerivatives) error {
	q, p := utils.GenerateInsertSQL(TableArtifactDerivatives, map[string]any{
		"d_id":               uuid.NewString(),
		"artifact_id":        a.ArtifactID,
		"kind":               a.Kind,
		"page_num":           a.PageNum,
		"storage":            a.Storage,
		"bucket":             a.Bucket,
		"object_key":         a.ObjectKey,
		"detected_mime_type": a.DetectedMIME,
		"byte_size":          a.ByteSize,
		"sha256_hex":         a.SHA256Hex,
		"created_at":         time.Now(),
	})
	_, err := r.db.Client().ExecContext(ctx, q, p...)
	return err
}

func (r *Repo) GetAllDerivativesForArtifact(ctx context.Context, artifactID uuid.UUID) ([]*entities.ArtifactDerivatives, error) {
	q := fmt.Sprintf("SELECT %s FROM %s WHERE artifact_id = $1", artifactDerivativesColumnsStr, TableArtifactDerivatives)
	return database.QueryRowsToStruct[entities.ArtifactDerivatives](ctx, r.db.Client(), q, artifactID)
}
