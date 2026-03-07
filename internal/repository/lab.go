package repository

import (
	"context"
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
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO lab_results (id, user_id, document_id, value, unit, reference_low, reference_high, flag, lab_name, collected_at, created_at)
		VALUES (:id, :user_id, :document_id, :value, :unit, :reference_low, :reference_high, :flag, :lab_name, :collected_at, :created_at)
		ON CONFLICT (user_id, LOWER(COALESCE(lab_name, '')), collected_at, value) DO NOTHING
	`, lab)
	return err
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
