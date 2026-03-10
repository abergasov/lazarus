package seed

import (
	"bytes"
	"lazarus/internal/entities"
	testhelpers "lazarus/internal/test_helpers"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func PreSeedDB(t *testing.T, cnt *testhelpers.TestContainer) *entities.Artifact {
	user := NewUserBuilder().PopulateTests(t, cnt)
	artifact := NewArtifactBuilder(user.ID).WithHash("16bf5128718952fbd0071983ee92e92045ae821ad2dc1f0f0ffbe5635cb84c20").PopulateTests(t, cnt)

	pwd, err := os.Getwd()
	require.NoError(t, err)
	objectPath := path.Join(strings.Split(pwd, "internal")[0], "internal/test_helpers/seed/sample.pdf")
	b, err := os.ReadFile(objectPath) //nolint:gosec // it ok
	require.NoError(t, err)
	require.NoError(t, cnt.BucketClient.Upload(cnt.Ctx, artifact.ObjectKey, bytes.NewReader(b), int64(len(b))))
	return artifact
}
