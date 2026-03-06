package entities

import (
	"time"

	"github.com/google/uuid"
)

type Medication struct {
	ID         uuid.UUID  `json:"id"          db:"id"`
	UserID     uuid.UUID  `json:"user_id"     db:"user_id"`
	RxCUI      string     `json:"rxcui"       db:"rxcui"`
	Name       string     `json:"name"        db:"name"`
	Dose       string     `json:"dose"        db:"dose"`
	Frequency  string     `json:"frequency"   db:"frequency"`
	Route      string     `json:"route"       db:"route"`
	Prescriber string     `json:"prescriber"  db:"prescriber"`
	IsActive   bool       `json:"is_active"   db:"is_active"`
	StartedAt  *time.Time `json:"started_at"  db:"started_at"`
	EndedAt    *time.Time `json:"ended_at"    db:"ended_at"`
	CreatedAt  time.Time  `json:"created_at"  db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"  db:"updated_at"`
}
