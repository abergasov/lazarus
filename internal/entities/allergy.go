package entities

import (
	"time"

	"github.com/google/uuid"
)

type Allergy struct {
	ID         uuid.UUID `json:"id"          db:"id"`
	UserID     uuid.UUID `json:"user_id"     db:"user_id"`
	Substance  string    `json:"substance"   db:"substance"`
	RxCUI      *string   `json:"rxcui"       db:"rxcui"`
	Severity   string    `json:"severity"    db:"severity"` // mild | moderate | severe | life_threatening
	Reaction   *string   `json:"reaction"    db:"reaction"`
	ReportedAt time.Time `json:"reported_at" db:"reported_at"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
}
