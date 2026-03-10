package artifact_parser_test

import (
	testhelpers "lazarus/internal/test_helpers"
	"lazarus/internal/test_helpers/seed"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactParser(t *testing.T) {
	testhelpers.SkipIfCI(t)

	// given
	testCfg := testhelpers.GetTestConfig(t)
	testCfg.S3.MaxUploadSizeBytes = 1024 * 1024
	container := testhelpers.GetCleanWithConfig(t, testCfg)
	artifact := seed.PreSeedDB(t, container)
	require.NoError(t, container.ServiceArtifactInspector.InspectArtifact(artifact))
	artifactDerivatives, err := container.Repo.GetAllDerivativesForArtifact(container.Ctx, artifact.ID)
	require.NoError(t, err)
	require.Len(t, artifactDerivatives, 5)

	// when
	require.NoError(t, container.ServiceArtifactParser.ProcessArtifact(artifact))

	// then
	labResults, err := container.Repo.GetLabResultByArtifactID(container.Ctx, artifact.ID)
	require.NoError(t, err)
	require.Len(t, labResults, 4)
	medications, err := container.Repo.GetAllMedicationsForUser(container.Ctx, artifact.OwnerID)
	require.NoError(t, err)
	require.Len(t, medications, 0)
}
