package entities

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type ArtifactDerivativeType string

const (
	ArtifactDerivativeTypeThumbnail    ArtifactDerivativeType = "thumbnail"
	ArtifactDerivativeTypePreview      ArtifactDerivativeType = "preview"
	ArtifactDerivativeTypePDFImagePage ArtifactDerivativeType = "pdf_page_image"
)

type ArtifactDerivatives struct {
	ID           uuid.UUID              `db:"d_id" json:"id"`
	ArtifactID   uuid.UUID              `db:"artifact_id" json:"artifact_id"`
	Kind         ArtifactDerivativeType `db:"kind" json:"kind"`
	PageNum      sql.NullInt32          `db:"page_num" json:"page_num,omitempty"`
	Storage      ArtifactStorage        `db:"storage" json:"storage"`
	Bucket       string                 `db:"bucket" json:"bucket"`
	ObjectKey    string                 `db:"object_key" json:"object_key"`
	DetectedMIME string                 `db:"detected_mime_type" json:"mime_type"`
	ByteSize     int64                  `db:"byte_size" json:"byte_size"`
	SHA256Hex    string                 `db:"sha256_hex" json:"sha256_hex"`
	CreatedAt    time.Time              `db:"created_at" json:"created_at"`
}
