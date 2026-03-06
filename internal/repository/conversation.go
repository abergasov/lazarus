package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
)

type ConversationRepo struct {
	db *sqlx.DB
}

func NewConversationRepo(db *sqlx.DB) *ConversationRepo {
	return &ConversationRepo{db: db}
}

func (r *ConversationRepo) Create(ctx context.Context, conv *entities.Conversation) error {
	conv.ID = uuid.New()
	conv.CreatedAt = time.Now()
	conv.UpdatedAt = conv.CreatedAt
	msgJSON, _ := json.Marshal(conv.Messages)
	if msgJSON == nil {
		msgJSON = []byte("[]")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO conversations (id, user_id, context_type, context_id, messages, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, conv.ID, conv.UserID, conv.ContextType, conv.ContextID, msgJSON, conv.CreatedAt, conv.UpdatedAt)
	return err
}

func (r *ConversationRepo) Get(ctx context.Context, id uuid.UUID) (*entities.Conversation, error) {
	var row struct {
		ID          uuid.UUID `db:"id"`
		UserID      uuid.UUID `db:"user_id"`
		ContextType string    `db:"context_type"`
		ContextID   string    `db:"context_id"`
		MsgJSON     []byte    `db:"messages"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	err := r.db.GetContext(ctx, &row, `SELECT id, user_id, context_type, context_id, messages, created_at, updated_at FROM conversations WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	conv := &entities.Conversation{
		ID: row.ID, UserID: row.UserID, ContextType: row.ContextType, ContextID: row.ContextID,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.MsgJSON != nil {
		_ = json.Unmarshal(row.MsgJSON, &conv.Messages)
	}
	if conv.Messages == nil {
		conv.Messages = []entities.ConversationMessage{}
	}
	return conv, nil
}

func (r *ConversationRepo) AppendMessage(ctx context.Context, id uuid.UUID, msg entities.ConversationMessage) error {
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE conversations
		SET messages = messages || $2::jsonb, updated_at = NOW()
		WHERE id = $1
	`, id, msgJSON)
	return err
}

func (r *ConversationRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]entities.Conversation, error) {
	var rows []struct {
		ID          uuid.UUID `db:"id"`
		UserID      uuid.UUID `db:"user_id"`
		ContextType string    `db:"context_type"`
		ContextID   string    `db:"context_id"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	err := r.db.SelectContext(ctx, &rows, `
		SELECT id, user_id, context_type, context_id, created_at, updated_at
		FROM conversations WHERE user_id = $1 ORDER BY updated_at DESC LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	convs := make([]entities.Conversation, len(rows))
	for i, row := range rows {
		convs[i] = entities.Conversation{
			ID: row.ID, UserID: row.UserID, ContextType: row.ContextType, ContextID: row.ContextID,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		}
	}
	return convs, nil
}

func (r *ConversationRepo) GetByContext(ctx context.Context, userID uuid.UUID, contextType, contextID string) (*entities.Conversation, error) {
	var row struct {
		ID          uuid.UUID `db:"id"`
		UserID      uuid.UUID `db:"user_id"`
		ContextType string    `db:"context_type"`
		ContextID   string    `db:"context_id"`
		MsgJSON     []byte    `db:"messages"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	err := r.db.GetContext(ctx, &row, `
		SELECT id, user_id, context_type, context_id, messages, created_at, updated_at
		FROM conversations WHERE user_id = $1 AND context_type = $2 AND context_id = $3
		ORDER BY updated_at DESC LIMIT 1
	`, userID, contextType, contextID)
	if err != nil {
		return nil, err
	}
	conv := &entities.Conversation{
		ID: row.ID, UserID: row.UserID, ContextType: row.ContextType, ContextID: row.ContextID,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.MsgJSON != nil {
		_ = json.Unmarshal(row.MsgJSON, &conv.Messages)
	}
	if conv.Messages == nil {
		conv.Messages = []entities.ConversationMessage{}
	}
	return conv, nil
}
