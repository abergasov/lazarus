package repository_test

import (
	"lazarus/internal/entities"
	testhelpers "lazarus/internal/test_helpers"
	"lazarus/internal/test_helpers/seed"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactCrud(t *testing.T) {
	// given
	container := testhelpers.GetClean(t)
	user := seed.NewUserBuilder().PopulateTests(t, container)

	// when
	artifact := seed.NewArtifactBuilder(user.ID).Build()
	artifactID := artifact.ID
	require.NoError(t, container.Repo.CreateArtifact(container.Ctx, artifactID, artifact))
	require.NotNil(t, artifact)

	// then
	artifactFromDB, err := container.Repo.GetArtifactByID(container.Ctx, artifact.OwnerID, artifactID)
	require.NoError(t, err)
	require.Equal(t, artifact.OriginalName, artifactFromDB.OriginalName)
	require.Equal(t, artifact.SHA256Hex, artifactFromDB.SHA256Hex)
	require.Equal(t, artifact.DeclaredMIME, artifactFromDB.DeclaredMIME)
	require.Equal(t, "", artifactFromDB.DetectedMIME)
	require.Equal(t, artifact.Status, artifactFromDB.Status)
	require.Equal(t, artifact.Kind, artifactFromDB.Kind)

	t.Run("should update artifact status", func(t *testing.T) {
		// when
		require.NoError(t, container.Repo.UpdateArtifactStatus(container.Ctx, artifactID, entities.ArtifactStatusClean))

		// then
		updatedArtifact, err := container.Repo.GetArtifactByID(container.Ctx, artifact.OwnerID, artifactID)
		require.NoError(t, err)
		require.Equal(t, entities.ArtifactStatusClean, updatedArtifact.Status)
		t.Run("should trigger error for wrong status", func(t *testing.T) {
			// when
			require.Error(t, container.Repo.UpdateArtifactStatus(container.Ctx, artifactID, "invalid_status"))

			// then
			updatedArtifact, err = container.Repo.GetArtifactByID(container.Ctx, artifact.OwnerID, artifactID)
			require.NoError(t, err)
			require.Equal(t, entities.ArtifactStatusClean, updatedArtifact.Status)
		})
	})
}
