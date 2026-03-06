package agent

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
	"lazarus/internal/repository"
)

type PatientModelStore struct {
	repo *repository.PatientModelRepo
}

func NewPatientModelStore(db *sqlx.DB) *PatientModelStore {
	return &PatientModelStore{repo: repository.NewPatientModelRepo(db)}
}

func (s *PatientModelStore) Load(ctx context.Context, userID uuid.UUID) (*entities.PatientModel, error) {
	return s.repo.Load(ctx, userID)
}

func (s *PatientModelStore) Save(ctx context.Context, model *entities.PatientModel) error {
	return s.repo.Save(ctx, model)
}
