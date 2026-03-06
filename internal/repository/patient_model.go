package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
)

type PatientModelRepo struct {
	db *sqlx.DB
}

func NewPatientModelRepo(db *sqlx.DB) *PatientModelRepo {
	return &PatientModelRepo{db: db}
}

func (r *PatientModelRepo) Load(ctx context.Context, userID uuid.UUID) (*entities.PatientModel, error) {
	var row struct {
		UserID  uuid.UUID `db:"user_id"`
		Version int       `db:"version"`
		Data    []byte    `db:"data"`
	}
	err := r.db.GetContext(ctx, &row, `SELECT user_id, version, data FROM patient_models WHERE user_id = $1`, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &entities.PatientModel{UserID: userID}, nil
		}
		return nil, err
	}
	var model entities.PatientModel
	if err := json.Unmarshal(row.Data, &model); err != nil {
		return nil, err
	}
	model.UserID = row.UserID
	model.Version = row.Version
	return &model, nil
}

func (r *PatientModelRepo) Save(ctx context.Context, model *entities.PatientModel) error {
	model.LastSynthesized = time.Now()
	data, err := json.Marshal(model)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO patient_models (user_id, version, data)
		VALUES ($1, 1, $2)
		ON CONFLICT (user_id) DO UPDATE
		SET version = patient_models.version + 1, data = $2, updated_at = NOW()
	`, model.UserID, data)
	return err
}
