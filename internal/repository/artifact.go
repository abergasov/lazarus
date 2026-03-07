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

const TableArtifacts = "artifacts"

var (
	artifactColumns = []string{
		"a_id",
		"owner_id",
		"kind",
		"status",
		"declared_mime_type",
		"detected_mime_type",
		"original_name",
		"byte_size",
		"sha256_hex",
		"storage",
		"bucket",
		"object_key",
		"created_at",
		"updated_at",
		"meta_json",
	}
	artifactColumnsStr = strings.Join(artifactColumns, ",")
)

func (r *Repo) CreateArtifact(ctx context.Context, a *entities.Artifact) (uuid.UUID, error) {
	result := uuid.New()
	q, p := utils.GenerateInsertSQL(TableArtifacts, map[string]any{
		"a_id":               result.String(),
		"owner_id":           a.OwnerID,
		"kind":               a.Kind,
		"status":             entities.ArtifactStatusQuarantined,
		"declared_mime_type": a.DeclaredMIME,
		"detected_mime_type": "",
		"original_name":      a.OriginalName,
		"byte_size":          a.ByteSize,
		"sha256_hex":         a.SHA256Hex,
		"storage":            a.Storage,
		"bucket":             a.Bucket,
		"object_key":         a.ObjectKey,
		"created_at":         time.Now(),
		"updated_at":         time.Now(),
		"meta_json":          a.MetaJSON,
	})
	_, err := r.db.Client().ExecContext(ctx, q, p...)
	return result, err
}

func (r *Repo) UpdateArtifactStatus(ctx context.Context, id uuid.UUID, st entities.ArtifactStatus) error {
	if !st.Valid() {
		return fmt.Errorf("invalid artifact status: %q", st)
	}
	q := fmt.Sprintf("UPDATE %s SET status = $1, updated_at = NOW() WHERE a_id = $2", TableArtifacts)
	_, err := r.db.Client().ExecContext(ctx, q, st, id)
	return err
}

func (r *Repo) GetAllArtifactsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*entities.Artifact, error) {
	q := fmt.Sprintf("SELECT %s FROM %s WHERE owner_id = $1 ORDER BY created_at DESC", artifactColumnsStr, TableArtifacts)
	return database.QueryRowsToStruct[entities.Artifact](ctx, r.db.Client(), q, ownerID)
}

func (r *Repo) GetArtifactByID(ctx context.Context, artifactID uuid.UUID) (*entities.Artifact, error) {
	q := fmt.Sprintf("SELECT %s FROM %s WHERE a_id = $1", artifactColumnsStr, TableArtifacts)
	return database.QueryRowToStruct[entities.Artifact](ctx, r.db.Client(), q, artifactID)
}
