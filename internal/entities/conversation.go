package entities

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID           uuid.UUID             `json:"id"                      db:"id"`
	UserID       uuid.UUID             `json:"-"                       db:"user_id"`
	ContextType  string                `json:"context_type"            db:"context_type"`
	ContextID    string                `json:"context_id"              db:"context_id"`
	Messages     []ConversationMessage `json:"messages,omitempty"      db:"-"`
	MsgJSON      []byte                `json:"-"                       db:"messages"`
	MessageCount int                   `json:"message_count,omitempty" db:"-"`
	CreatedAt    time.Time             `json:"created_at"              db:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"              db:"updated_at"`
}
