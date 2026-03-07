package entities

import (
	"time"

	"github.com/google/uuid"
)

type Question struct {
	ID        uuid.UUID  `json:"id"         db:"id"`
	UserID    uuid.UUID  `json:"user_id"    db:"user_id"`
	VisitID   *uuid.UUID `json:"visit_id"   db:"visit_id"`
	Text      string     `json:"text"       db:"text"`
	Rationale string     `json:"rationale"  db:"rationale"`
	Urgency   string     `json:"urgency"    db:"urgency"`
	Source    string     `json:"source"     db:"source"`
	Asked     bool       `json:"asked"      db:"asked"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}
