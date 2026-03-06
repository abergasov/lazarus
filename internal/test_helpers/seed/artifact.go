package seed

import (
	"context"
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
			OwnerID:      userID,
			Kind:         entities.ArtifactKindPDF,
			Status:       entities.ArtifactStatusQuarantined,
			DeclaredMIME: "application/pdf",
			DetectedMIME: "application/pdf",
			OriginalName: uuid.NewString()[:8] + ".pdf",
			ByteSize:     4832,
			SHA256Hex:    utils.HashSHA256([]byte(uuid.NewString())),
			Storage:      "",
			Bucket:       "",
			ObjectKey:    "",
			MetaJSON:     nil,
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
	artefactID, err := cnt.Repo.CreateArtifact(ctx, b.Build())
	require.NoError(t, err)
	artifact, err := cnt.Repo.GetArtefactByID(ctx, artefactID)
	require.NoError(t, err)
	require.NotNil(t, artifact)
	return artifact
}
