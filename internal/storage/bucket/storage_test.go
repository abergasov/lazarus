package bucket_test

import (
	"bytes"
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
	require.NoError(t, container.BucketClient.Upload(container.Ctx, key, bytes.NewReader(payload), int64(len(payload))))

	// then
	got, err := container.BucketClient.DownloadBytes(container.Ctx, key)
	require.NoError(t, err)
	require.Equal(t, payload, got)

	t.Run("should delete object", func(t *testing.T) {
		// when
		require.NoError(t, container.BucketClient.Delete(container.Ctx, key))

		// then
		_, err = container.BucketClient.Download(container.Ctx, key)
		require.Error(t, err)
	})
}
