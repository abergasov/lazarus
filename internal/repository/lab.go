package repository

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
)

type LabRepo struct {
	db *sqlx.DB
}

func NewLabRepo(db *sqlx.DB) *LabRepo {
	return &LabRepo{db: db}
}

func (r *LabRepo) Insert(ctx context.Context, lab *entities.LabResult) error {
	lab.ID = uuid.New()
	lab.CreatedAt = time.Now()
	normalized := NormalizeLabName(lab.LabName)
	lab.NormalizedName = &normalized
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO lab_results (id, user_id, document_id, value, unit, reference_low, reference_high, flag, lab_name, normalized_name, collected_at, created_at)
		VALUES (:id, :user_id, :document_id, :value, :unit, :reference_low, :reference_high, :flag, :lab_name, :normalized_name, :collected_at, :created_at)
		ON CONFLICT (user_id, LOWER(COALESCE(normalized_name, COALESCE(lab_name, ''))), collected_at, value) DO NOTHING
	`, lab)
	return err
}

var reParenthetical = regexp.MustCompile(`\s*\([^)]*\)\s*`)
var reWhitespace = regexp.MustCompile(`\s+`)

// NormalizeLabName strips parenthetical aliases, common prefixes, and normalizes whitespace.
func NormalizeLabName(name *string) string {
	if name == nil || *name == "" {
		return ""
	}
	n := strings.ToLower(*name)
	// Remove parenthetical content: "ALB (Albumin)" → "alb"
	n = reParenthetical.ReplaceAllString(n, " ")
	// Remove common non-informative prefixes
	for _, prefix := range []string{"serum ", "plasma ", "blood ", "total "} {
		n = strings.TrimPrefix(n, prefix)
	}
	n = reWhitespace.ReplaceAllString(strings.TrimSpace(n), " ")
	return n
}

func (r *LabRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]entities.LabResult, error) {
	var labs []entities.LabResult
	err := r.db.SelectContext(ctx, &labs, `
		SELECT * FROM lab_results
		WHERE user_id = $1
		ORDER BY lab_name, collected_at DESC
	`, userID)
	return labs, err
}

func (r *LabRepo) ListByDocument(ctx context.Context, userID, documentID uuid.UUID) ([]entities.LabResult, error) {
	var labs []entities.LabResult
	err := r.db.SelectContext(ctx, &labs, `
		SELECT * FROM lab_results
		WHERE user_id = $1 AND document_id = $2
		ORDER BY lab_name, collected_at DESC
	`, userID, documentID)
	return labs, err
}

func (r *LabRepo) Update(ctx context.Context, lab *entities.LabResult) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE lab_results SET lab_name = $1, value = $2, unit = $3, flag = $4, collected_at = $5
		WHERE id = $6 AND user_id = $7
	`, lab.LabName, lab.Value, lab.Unit, lab.Flag, lab.CollectedAt, lab.ID, lab.UserID)
	return err
}

func (r *LabRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM lab_results WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *LabRepo) GetTrend(ctx context.Context, userID uuid.UUID, loincCode string, months int) ([]entities.DataPoint, error) {
	var pts []entities.DataPoint
	err := r.db.SelectContext(ctx, &pts, `
		SELECT value, collected_at, flag
		FROM lab_results
		WHERE user_id = $1 AND loinc_code = $2
		  AND collected_at > NOW() - ($3 || ' months')::interval
		ORDER BY collected_at ASC
	`, userID, loincCode, months)
	return pts, err
}
