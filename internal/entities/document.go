package entities

import (
	"time"

	"github.com/google/uuid"
)

type Document struct {
	ID           uuid.UUID  `json:"id"            db:"id"`
	UserID       uuid.UUID  `json:"user_id"       db:"user_id"`
	VisitID      *uuid.UUID `json:"visit_id"      db:"visit_id"`
	StorageKey   string     `json:"storage_key"   db:"storage_key"`
	MimeType     *string    `json:"mime_type"     db:"mime_type"`
	FileName     *string    `json:"file_name"     db:"file_name"`
	SizeBytes    *int64     `json:"size_bytes"    db:"size_bytes"`
	SourceName   *string    `json:"source_name"   db:"source_name"`
	SourceType   string     `json:"source_type"   db:"source_type"`
	Category     string     `json:"category"      db:"category"`
	Specialty    *string    `json:"specialty"     db:"specialty"`
	Summary      *string    `json:"summary"       db:"summary"`
	DocumentDate *time.Time `json:"document_date" db:"document_date"`
	ParseStatus  string     `json:"parse_status"  db:"parse_status"`
	ParsedAt     *time.Time `json:"parsed_at"     db:"parsed_at"`
	CreatedAt    time.Time  `json:"created_at"    db:"created_at"`
}

const (
	ParseStatusPending    = "pending"
	ParseStatusProcessing = "processing"
	ParseStatusDone       = "done"
	ParseStatusFailed     = "failed"
)
