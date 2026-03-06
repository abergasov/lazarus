package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
)

type MedicationRepo struct {
	db *sqlx.DB
}

func NewMedicationRepo(db *sqlx.DB) *MedicationRepo {
	return &MedicationRepo{db: db}
}

func (r *MedicationRepo) Create(ctx context.Context, med *entities.Medication) error {
	med.ID = uuid.New()
	med.CreatedAt = time.Now()
	med.UpdatedAt = time.Now()
	med.IsActive = true
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO medications (id, user_id, rxcui, name, dose, frequency, route, prescriber, is_active, started_at, created_at, updated_at)
		VALUES (:id, :user_id, :rxcui, :name, :dose, :frequency, :route, :prescriber, :is_active, :started_at, :created_at, :updated_at)
	`, med)
	return err
}

func (r *MedicationRepo) ListActive(ctx context.Context, userID uuid.UUID) ([]entities.Medication, error) {
	var meds []entities.Medication
	err := r.db.SelectContext(ctx, &meds, `
		SELECT * FROM medications WHERE user_id = $1 AND is_active = TRUE ORDER BY created_at DESC
	`, userID)
	return meds, err
}

func (r *MedicationRepo) ListAll(ctx context.Context, userID uuid.UUID) ([]entities.Medication, error) {
	var meds []entities.Medication
	err := r.db.SelectContext(ctx, &meds, `
		SELECT * FROM medications WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	return meds, err
}

func (r *MedicationRepo) Deactivate(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE medications SET is_active = FALSE, ended_at = $1, updated_at = $1 WHERE id = $2 AND user_id = $3`,
		now, id, userID)
	return err
}
