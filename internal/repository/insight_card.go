package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
)

type InsightCardRepo struct {
	db *sqlx.DB
}

func NewInsightCardRepo(db *sqlx.DB) *InsightCardRepo {
	return &InsightCardRepo{db: db}
}

func (r *InsightCardRepo) Create(ctx context.Context, card *entities.InsightCard) error {
	actionsJSON, err := json.Marshal(card.Actions)
	if err != nil {
		actionsJSON = []byte("[]")
	}
	card.ID = uuid.New()
	card.CreatedAt = time.Now()
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO insight_cards (id, user_id, type, title, body, severity, context_type, context_id, actions, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, card.ID, card.UserID, card.Type, card.Title, card.Body, card.Severity, card.ContextType, card.ContextID, actionsJSON, card.CreatedAt)
	return err
}

func (r *InsightCardRepo) ListActive(ctx context.Context, userID uuid.UUID) ([]entities.InsightCard, error) {
	rows := []struct {
		entities.InsightCard
		ActionsRaw []byte `db:"actions"`
	}{}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, user_id, type, title, body, severity, context_type, context_id, actions, dismissed_at, created_at
		FROM insight_cards WHERE user_id = $1 AND dismissed_at IS NULL
		ORDER BY created_at DESC LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	cards := make([]entities.InsightCard, len(rows))
	for i, row := range rows {
		cards[i] = row.InsightCard
		if row.ActionsRaw != nil {
			_ = json.Unmarshal(row.ActionsRaw, &cards[i].Actions)
		}
		if cards[i].Actions == nil {
			cards[i].Actions = []entities.Action{}
		}
	}
	return cards, nil
}

func (r *InsightCardRepo) Dismiss(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE insight_cards SET dismissed_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *InsightCardRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.InsightCard, error) {
	var row struct {
		entities.InsightCard
		ActionsRaw []byte `db:"actions"`
	}
	err := r.db.GetContext(ctx, &row, `
		SELECT id, user_id, type, title, body, severity, context_type, context_id, actions, dismissed_at, created_at
		FROM insight_cards WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	card := row.InsightCard
	if row.ActionsRaw != nil {
		_ = json.Unmarshal(row.ActionsRaw, &card.Actions)
	}
	if card.Actions == nil {
		card.Actions = []entities.Action{}
	}
	return &card, nil
}
