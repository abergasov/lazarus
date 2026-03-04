package entities

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type (
	ArtifactStorage string
	ArtifactKind    string
)

const (
	ArtifactKindPDF          ArtifactKind = "pdf"
	ArtifactKindImage        ArtifactKind = "image"
	ArtifactKindScan         ArtifactKind = "scan"
	ArtifactKindLab          ArtifactKind = "lab"
	ArtifactKindPrescription ArtifactKind = "prescription"
	ArtifactKindReferral     ArtifactKind = "referral"
	ArtifactKindInvoice      ArtifactKind = "invoice"
	ArtifactKindOther        ArtifactKind = "other"

	ArtifactStorageS3 ArtifactStorage = "s3"
)

func (k ArtifactKind) Valid() bool {
	artifactMap := map[ArtifactKind]struct{}{
		ArtifactKindPDF:          {},
		ArtifactKindImage:        {},
		ArtifactKindScan:         {},
		ArtifactKindLab:          {},
		ArtifactKindPrescription: {},
		ArtifactKindReferral:     {},
		ArtifactKindInvoice:      {},
		ArtifactKindOther:        {},
	}
	_, ok := artifactMap[k]
	return ok
}

func (s ArtifactStorage) Valid() bool {
	storageMap := map[ArtifactStorage]struct{}{
		ArtifactStorageS3: {},
	}
	_, ok := storageMap[s]
	return ok
}

// Artifact is immutable (except soft fields like tags/notes if you add them).
// Raw bytes live in object storage.
type Artifact struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	OwnerID   uuid.UUID       `db:"owner_id" json:"owner_id"`
	Kind      ArtifactKind    `db:"kind" json:"kind"`
	MimeType  string          `db:"mime_type" json:"mime_type"`
	ByteSize  int64           `db:"byte_size" json:"byte_size"`
	SHA256Hex string          `db:"sha256_hex" json:"sha256_hex"` // lowercase hex, 64 chars
	Storage   ArtifactStorage `db:"storage" json:"storage"`
	Bucket    string          `db:"bucket" json:"bucket"`
	ObjectKey string          `db:"object_key" json:"object_key"` // recommend: "sha256/<first2>/<sha>"
	CreatedAt time.Time       `db:"created_at" json:"created_at"`

	// Optional: external metadata (client filename, source system, etc.)
	MetaJSON json.RawMessage `db:"meta_json" json:"meta_json"`
}

func (a *Artifact) Validate() error {
	if a.ID == uuid.Nil {
		return errors.New("artifact.id is empty")
	}
	if a.OwnerID == uuid.Nil {
		return errors.New("artifact.owner_id is empty")
	}
	if !a.Kind.Valid() {
		return fmt.Errorf("artifact.kind invalid: %q", a.Kind)
	}
	if a.ByteSize < 0 {
		return errors.New("artifact.byte_size negative")
	}
	if len(a.MimeType) > 255 {
		return errors.New("artifact.mime_type too long")
	}
	if len(a.SHA256Hex) != 64 {
		return errors.New("artifact.sha256_hex must be 64 hex chars")
	}
	if !a.Storage.Valid() {
		return fmt.Errorf("artifact.storage invalid: %q", a.Storage)
	}
	if a.Bucket == "" || len(a.Bucket) > 255 {
		return errors.New("artifact.bucket invalid")
	}
	if a.ObjectKey == "" || len(a.ObjectKey) > 1024 {
		return errors.New("artifact.object_key invalid")
	}
	if len(a.MetaJSON) > 0 && !json.Valid(a.MetaJSON) {
		return errors.New("artifact.meta_json invalid json")
	}
	return nil
}
