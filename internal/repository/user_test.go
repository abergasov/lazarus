package repository_test

import (
	testhelpers "lazarus/internal/test_helpers"
	"lazarus/internal/test_helpers/seed"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserCRUD(t *testing.T) {
	// given
	container := testhelpers.GetClean(t)
	t.Run("no users on start", func(t *testing.T) {
		users, err := container.Repo.GetAllUsers(container.Ctx)
		require.NoError(t, err)
		require.Len(t, users, 0)
	})
	t.Run("should add user", func(t *testing.T) {
		// given
		googleUser := seed.NewUserBuilder().Build()

		// when
		require.NoError(t, container.Repo.AddGoogleUser(container.Ctx, googleUser))

		// then
		usr, err := container.Repo.GetUserByMail(container.Ctx, googleUser.Email)
		require.NoError(t, err)
		require.NotNil(t, usr)
		require.Equal(t, googleUser.Email, usr.Email)
		require.Equal(t, googleUser.Name, usr.UserName)
		require.NotZero(t, usr.ID)
		require.NotZero(t, usr.CreatedAt)
		require.NotZero(t, usr.UpdatedAt)

		t.Run("should get user by id", func(t *testing.T) {
			res, err := container.Repo.GetUserByID(container.Ctx, usr.ID)
			require.NoError(t, err)
			require.NotNil(t, res)
			require.Equal(t, usr.Email, res.Email)
			require.Equal(t, usr.UserName, res.UserName)
			require.NotZero(t, res.ID)
			require.NotZero(t, res.CreatedAt)
			require.NotZero(t, res.UpdatedAt)
		})
	})
}
