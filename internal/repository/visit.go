package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
)

type VisitRepo struct {
	db *sqlx.DB
}

func NewVisitRepo(db *sqlx.DB) *VisitRepo {
	return &VisitRepo{db: db}
}

func (r *VisitRepo) Create(ctx context.Context, v *entities.Visit) error {
	v.ID = uuid.New()
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()
	if v.Status == "" {
		v.Status = entities.VisitStatusPreparing
	}
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO visits (id, user_id, doctor_name, specialty, clinic_name, visit_date, visit_type, reason, status, plan_json, outcome_json, follow_up_date, created_at, updated_at)
		VALUES (:id, :user_id, :doctor_name, :specialty, :clinic_name, :visit_date, :visit_type, :reason, :status, :plan_json, :outcome_json, :follow_up_date, :created_at, :updated_at)
	`, v)
	return err
}

func (r *VisitRepo) Get(ctx context.Context, id string) (*entities.Visit, error) {
	var v entities.Visit
	err := r.db.GetContext(ctx, &v, `SELECT * FROM visits WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("get visit: %w", err)
	}
	return &v, nil
}

func (r *VisitRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]entities.Visit, error) {
	var visits []entities.Visit
	err := r.db.SelectContext(ctx, &visits, `
		SELECT * FROM visits WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	return visits, err
}

func (r *VisitRepo) UpdatePhase(ctx context.Context, id string, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE visits SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id)
	return err
}

func (r *VisitRepo) UpdatePlan(ctx context.Context, id string, planJSON []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE visits SET plan_json = $1, updated_at = NOW() WHERE id = $2`,
		planJSON, id)
	return err
}

func (r *VisitRepo) UpdateOutcome(ctx context.Context, id string, outcomeJSON []byte) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE visits SET outcome_json = $1, updated_at = NOW() WHERE id = $2`,
		outcomeJSON, id)
	return err
}

func (r *VisitRepo) AppendNote(ctx context.Context, id string, note entities.VisitNote) error {
	noteJSON, err := json.Marshal(note)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx,
		`UPDATE visits SET notes_json = COALESCE(notes_json, '[]'::jsonb) || $1::jsonb, updated_at = NOW() WHERE id = $2`,
		noteJSON, id)
	return err
}

func (r *VisitRepo) Delete(ctx context.Context, id string) error {
	// Clean up related data first
	_, _ = r.db.ExecContext(ctx, `DELETE FROM agent_sessions WHERE visit_id = $1`, id)
	_, _ = r.db.ExecContext(ctx, `UPDATE documents SET visit_id = NULL WHERE visit_id = $1`, id)
	_, _ = r.db.ExecContext(ctx, `DELETE FROM conversations WHERE context_type = 'visit' AND context_id = $1`, id)
	_, err := r.db.ExecContext(ctx, `DELETE FROM visits WHERE id = $1`, id)
	return err
}

func (r *VisitRepo) FindUpcomingUnprepared(ctx context.Context, within time.Duration) ([]entities.Visit, error) {
	var visits []entities.Visit
	err := r.db.SelectContext(ctx, &visits, `
		SELECT v.* FROM visits v
		WHERE v.status = 'preparing'
		  AND v.visit_date BETWEEN NOW() AND NOW() + $1::interval
		  AND NOT EXISTS (
			SELECT 1 FROM agent_sessions s WHERE s.visit_id = v.id AND s.phase = 'preparing'
		  )
	`, fmt.Sprintf("%.0f seconds", within.Seconds()))
	return visits, err
}
