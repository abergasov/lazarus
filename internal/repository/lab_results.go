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

const TableLabResult = "lab_results"

var (
	tableLabResultsCols = []string{
		"id",
		"user_id",
		"document_id",
		"loinc_code",
		"lab_value",
		"unit",
		"reference_low",
		"reference_high",
		"flag",
		"lab_name",
		"collected_at",
		"normalized_name",
		"created_at",
	}
	tableLabResultsColsStr = strings.Join(tableLabResultsCols, ",")
)

func (r *Repo) InsertLabResult(ctx context.Context, l *entities.LabResult) (uuid.UUID, error) {
	resID := uuid.New()
	q, p := utils.GenerateInsertSQL(TableLabResult, map[string]any{
		"id":              resID,
		"user_id":         l.UserID,
		"document_id":     l.DocumentID,
		"loinc_code":      l.LoincCode,
		"lab_value":       l.Value,
		"unit":            l.Unit,
		"reference_low":   l.ReferenceLow,
		"reference_high":  l.ReferenceHigh,
		"flag":            l.Flag,
		"lab_name":        l.LabName,
		"collected_at":    l.CollectedAt,
		"normalized_name": l.NormalizedName,
		"created_at":      time.Now(),
	})
	q += " ON CONFLICT (user_id, LOWER(COALESCE(normalized_name, COALESCE(lab_name, ''))), collected_at, lab_value) DO NOTHING"
	_, err := r.db.Client().ExecContext(ctx, q, p...)
	return resID, err
}

func (r *Repo) GetLabResultByArtifactID(ctx context.Context, artifactID uuid.UUID) ([]*entities.LabResult, error) {
	q := fmt.Sprintf("SELECT %s FROM %s WHERE document_id = $1", tableLabResultsColsStr, TableLabResult)
	return database.QueryRowsToStruct[entities.LabResult](ctx, r.db.Client(), q, artifactID)
}
