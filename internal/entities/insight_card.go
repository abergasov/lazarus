package entities

import (
	"time"

	"github.com/google/uuid"
)

type InsightCard struct {
	ID          uuid.UUID  `json:"id"           db:"id"`
	UserID      uuid.UUID  `json:"-"            db:"user_id"`
	Type        string     `json:"type"         db:"type"`
	Title       string     `json:"title"        db:"title"`
	Body        string     `json:"body"         db:"body"`
	Severity    string     `json:"severity"     db:"severity"`
	ContextType string     `json:"context_type" db:"context_type"`
	ContextID   string     `json:"context_id"   db:"context_id"`
	Actions     []Action   `json:"actions"      db:"-"`
	ActionsJSON []byte     `json:"-"            db:"actions"`
	DismissedAt *time.Time `json:"dismissed_at" db:"dismissed_at"`
	CreatedAt   time.Time  `json:"created_at"   db:"created_at"`
}

type Action struct {
	Label    string `json:"label"`
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
	Body     string `json:"body,omitempty"`
}

const (
	InsightLabTrend     = "lab_trend"
	InsightRiskChange   = "risk_change"
	InsightGap          = "gap"
	InsightVisitPrep    = "visit_prep"
	InsightDocProcessed = "document_processed"
	InsightWelcome      = "welcome"

	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityUrgent  = "urgent"
)
