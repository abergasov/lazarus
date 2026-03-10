package artifact_inspector_test

import (
	testhelpers "lazarus/internal/test_helpers"
	"lazarus/internal/test_helpers/seed"
	"lazarus/internal/utils"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactInspectorService(t *testing.T) {
	// given
	testCfg := testhelpers.GetTestConfig(t)
	testCfg.S3.MaxUploadSizeBytes = 1024 * 1024
	container := testhelpers.GetCleanWithConfig(t, testCfg)
	artifact := seed.PreSeedDB(t, container)

	// when
	require.NoError(t, container.ServiceArtifactInspector.InspectArtifact(artifact))

	// then
	derivatives, err := container.Repo.GetAllDerivativesForArtifact(container.Ctx, artifact.ID)
	require.NoError(t, err)
	require.Len(t, derivatives, 5)
	require.Equal(t, "d6cbbf668a2daf3c103c76f793422437b442af6922e1c7372b7caff75968e5de", derivatives[0].SHA256Hex)
	require.Equal(t, "db8a9169247a66937b3eb938c3c8d916b15ab0877410cd1db636ca0078d5cb81", derivatives[1].SHA256Hex)
	require.Equal(t, "d1d072132966cbe71eebcf45d4cfe4042237d29c71f7acbd436ab5b3ad3d71b7", derivatives[2].SHA256Hex)
	require.Equal(t, "a2e2819e64643f1d0c01481b214c66e7ed3aa908b5ce28d02db161be4d88deb5", derivatives[3].SHA256Hex)
	require.Equal(t, "2f85659bb77009b7432683c59cbac400329d59e5a35fe1031f446b74ad8e5027", derivatives[4].SHA256Hex)
	t.Run("verify hashses from storage", func(t *testing.T) {
		for _, d := range derivatives {
			rawBytes, errD := container.BucketClient.DownloadBytes(container.Ctx, d.ObjectKey)
			require.NoError(t, errD)

			require.Equal(t, utils.HashSHA256(rawBytes), d.SHA256Hex)
		}
	})
}
