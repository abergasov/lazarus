package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"lazarus/internal/entities"
	testhelpers "lazarus/internal/test_helpers"
	"lazarus/internal/utils"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type ArtifactBuilder struct {
	artifact *entities.Artifact
}

func NewArtifactBuilder(userID uuid.UUID) *ArtifactBuilder {
	return &ArtifactBuilder{
		artifact: &entities.Artifact{
			ID:           uuid.New(),
			OwnerID:      userID,
			Kind:         entities.ArtifactKindPDF,
			Status:       entities.ArtifactStatusQuarantined,
			DeclaredMIME: "application/pdf",
			DetectedMIME: "application/pdf",
			OriginalName: uuid.NewString()[:8] + ".pdf",
			ByteSize:     4832,
			SHA256Hex:    utils.HashSHA256([]byte(uuid.NewString())),
			Storage:      entities.ArtifactStorageS3,
			Bucket:       uuid.NewString()[:8],
			ObjectKey:    uuid.NewString()[:8],
			MetaJSON:     sql.Null[json.RawMessage]{},
		},
	}
}

func (b *ArtifactBuilder) WithKind(kind entities.ArtifactKind) *ArtifactBuilder {
	b.artifact.Kind = kind
	return b
}

func (b *ArtifactBuilder) WithStatus(status entities.ArtifactStatus) *ArtifactBuilder {
	b.artifact.Status = status
	return b
}

func (b *ArtifactBuilder) Build() *entities.Artifact {
	return b.artifact
}

func (b *ArtifactBuilder) PopulateTests(t *testing.T, cnt *testhelpers.TestContainer) *entities.Artifact {
	ctx, cancel := context.WithTimeout(cnt.Ctx, 10*time.Second)
	defer cancel()
	artifact := b.Build()
	require.NoError(t, cnt.Repo.CreateArtifact(ctx, artifact))
	artifactDB, err := cnt.Repo.GetArtifactByID(ctx, artifact.OwnerID, artifact.ID)
	require.NoError(t, err)
	require.NotNil(t, artifactDB)
	return artifactDB
}
