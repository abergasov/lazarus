package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
	"lazarus/internal/provider"
)

// Session is the in-memory representation used during a single request
type Session struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	VisitID  uuid.UUID
	Phase    string
	Provider string
	Model    string
	Messages []provider.Message
}

type SessionStore struct {
	db *sqlx.DB
}

func NewSessionStore(db *sqlx.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) GetOrCreate(ctx context.Context, userID uuid.UUID, visitIDStr string, phase string) (*Session, error) {
	visitID := uuid.Nil
	if visitIDStr != "" {
		var err error
		visitID, err = uuid.Parse(visitIDStr)
		if err != nil {
			visitID = uuid.Nil
		}
	}

	// Try to find existing session for this visit+phase
	if visitID != uuid.Nil {
		var row struct {
			ID         string `db:"id"`
			ProviderID string `db:"provider_id"`
			ModelID    string `db:"model_id"`
			Messages   []byte `db:"messages"`
			Phase      string `db:"phase"`
		}
		err := s.db.GetContext(ctx, &row, `
			SELECT id, provider_id, model_id, messages, phase
			FROM agent_sessions
			WHERE visit_id = $1 AND phase = $2
			ORDER BY created_at DESC LIMIT 1
		`, visitID, phase)
		if err == nil {
			sessionID, _ := uuid.Parse(row.ID)
			sess := &Session{
				ID:       sessionID,
				UserID:   userID,
				VisitID:  visitID,
				Phase:    row.Phase,
				Provider: row.ProviderID,
				Model:    row.ModelID,
			}
			var msgs []entities.ConversationMessage
			if err := json.Unmarshal(row.Messages, &msgs); err == nil {
				for _, m := range msgs {
					sess.Messages = append(sess.Messages, provider.Message{
						Role:    m.Role,
						Content: m.Content,
					})
				}
			}
			return sess, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}

	return &Session{
		ID:      uuid.New(),
		UserID:  userID,
		VisitID: visitID,
		Phase:   phase,
	}, nil
}

func (s *SessionStore) Save(ctx context.Context, sess *Session) error {
	msgs := make([]entities.ConversationMessage, 0, len(sess.Messages))
	for _, m := range sess.Messages {
		msgs = append(msgs, entities.ConversationMessage{
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: time.Now(),
		})
	}
	data, err := json.Marshal(msgs)
	if err != nil {
		return err
	}

	visitID := interface{}(nil)
	if sess.VisitID != uuid.Nil {
		visitID = sess.VisitID
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO agent_sessions (id, user_id, visit_id, phase, provider_id, model_id, messages, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (id) DO UPDATE
		SET messages = $7, updated_at = NOW()
	`, sess.ID, sess.UserID, visitID, sess.Phase, sess.Provider, sess.Model, data)
	return err
}
