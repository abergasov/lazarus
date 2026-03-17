package entities

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type AuditEventType string

const (
	EventArtifactUploaded AuditEventType = "artifact_uploaded"
	EventArtifactDeleted  AuditEventType = "artifact_deleted" // logical delete if you add it
)

type AuditEvent struct {
	ID          uuid.UUID      `db:"id" json:"id"`
	At          time.Time      `db:"at" json:"at"`
	ActorUserID uuid.UUID      `db:"actor_user_id" json:"actor_user_id"` // who did it
	Type        AuditEventType `db:"type" json:"type"`

	ArtifactID *uuid.UUID      `db:"artifact_id" json:"artifact_id,omitempty"`
	DataJSON   json.RawMessage `db:"data_json" json:"data_json"`
}

func (e *AuditEvent) Validate() error {
	if e.ID == uuid.Nil {
		return errors.New("event.id is empty")
	}
	if e.ActorUserID == uuid.Nil {
		return errors.New("event.actor_user_id is empty")
	}
	if e.Type == "" {
		return errors.New("event.type is empty")
	}
	if len(e.DataJSON) > 0 && !json.Valid(e.DataJSON) {
		return errors.New("event.data_json invalid json")
	}
	return nil
}
