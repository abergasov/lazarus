package entities

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type (
	ArtifactStorage string
	ArtifactKind    string
	ArtifactStatus  string
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

	ArtifactStatusQuarantined ArtifactStatus = "QUARANTINED"
	ArtifactStatusClean       ArtifactStatus = "CLEAN"
	ArtifactStatusRejected    ArtifactStatus = "REJECTED"
)

func (k ArtifactStatus) Valid() bool {
	statusMap := map[ArtifactStatus]struct{}{
		ArtifactStatusQuarantined: {},
		ArtifactStatusClean:       {},
		ArtifactStatusRejected:    {},
	}
	_, ok := statusMap[k]
	return ok
}

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
	ID             uuid.UUID                 `db:"a_id" json:"id"`
	OwnerID        uuid.UUID                 `db:"owner_id" json:"owner_id"`
	Kind           ArtifactKind              `db:"kind" json:"kind"`
	Status         ArtifactStatus            `db:"status" json:"status"`
	DeclaredMIME   string                    `db:"declared_mime_type" json:"declared_mime_type"`
	DetectedMIME   string                    `db:"detected_mime_type" json:"detected_mime_type"`
	OriginalName   string                    `db:"original_name" json:"original_name"`
	ByteSize       int64                     `db:"byte_size" json:"byte_size"`
	SHA256Hex      string                    `db:"sha256_hex" json:"sha256_hex"`
	Storage        ArtifactStorage           `db:"storage" json:"storage"`
	Bucket         string                    `db:"bucket" json:"bucket"`
	ObjectKey      string                    `db:"object_key" json:"object_key"` // recommend: "sha256/<first2>/<sha>"
	CreatedAt      time.Time                 `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time                 `db:"updated_at" json:"updated_at"`
	ContentSummary string                    `db:"content_summary" json:"content_summary"`
	MetaJSON       sql.Null[json.RawMessage] `db:"meta_json" json:"meta_json"`
}
