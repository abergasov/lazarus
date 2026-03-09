package artifact_inspector_test

import (
	"bytes"
	testhelpers "lazarus/internal/test_helpers"
	"lazarus/internal/test_helpers/seed"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactInspectorService(t *testing.T) {
	// given
	container := testhelpers.GetClean(t)
	user := seed.NewUserBuilder().PopulateTests(t, container)
	artifact := seed.NewArtifactBuilder(user.ID).WithHash("16bf5128718952fbd0071983ee92e92045ae821ad2dc1f0f0ffbe5635cb84c20").PopulateTests(t, container)
	b, err := os.ReadFile("test_data/sample.pdf")
	require.NoError(t, err)
	require.NoError(t, container.BucketClient.Upload(container.Ctx, artifact.ObjectKey, bytes.NewReader(b), int64(len(b))))

	// when
	require.NoError(t, container.ServiceArtifactInspector.InspectArtifact(artifact))
}
