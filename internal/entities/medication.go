package entities

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Medication struct {
	ID         uuid.UUID      `json:"id"          db:"id"`
	UserID     uuid.UUID      `json:"user_id"     db:"user_id"`
	RxCUI      sql.NullString `json:"rxcui"       db:"rxcui"`
	Name       string         `json:"name"        db:"name"`
	Dose       string         `json:"dose"        db:"dose"`
	Frequency  string         `json:"frequency"   db:"frequency"`
	Route      sql.NullString `json:"route"       db:"route"`
	Prescriber sql.NullString `json:"prescriber"  db:"prescriber"`
	IsActive   bool           `json:"is_active"   db:"is_active"`
	StartedAt  sql.NullTime   `json:"started_at"  db:"started_at"`
	EndedAt    sql.NullTime   `json:"ended_at"    db:"ended_at"`
	CreatedAt  time.Time      `json:"created_at"  db:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"  db:"updated_at"`
}
