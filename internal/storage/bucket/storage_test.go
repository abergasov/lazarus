package bucket_test

import (
	"bytes"
	"io"
	testhelpers "lazarus/internal/test_helpers"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestClientCRUD(t *testing.T) {
	// given
	container := testhelpers.GetClean(t)
	payload := []byte(strings.Join([]string{
		uuid.NewString(),
		uuid.NewString(),
		uuid.NewString(),
		uuid.NewString(),
		uuid.NewString(),
	}, "\n"))
	key := uuid.NewString()

	// when
	require.NoError(t, container.BucketClient.Upload(container.Ctx, key, bytes.NewReader(payload)))

	// then
	rc, err := container.BucketClient.Download(container.Ctx, key)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, rc.Close())
	})
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}
