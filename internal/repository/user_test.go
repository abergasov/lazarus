package repository_test

import (
	"lazarus/internal/entities"
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
		compareUsers(t, googleUser, usr)

		t.Run("should get user by id", func(t *testing.T) {
			// when, then
			res, err := container.Repo.GetUserByID(container.Ctx, usr.ID)
			require.NoError(t, err)
			compareUsers(t, googleUser, res)
		})
	})
}

func compareUsers(t *testing.T, srcUser *entities.GoogleUser, targetUser *entities.User) {
	t.Helper()

	require.NotNil(t, targetUser)
	require.Equal(t, srcUser.Email, targetUser.Email)
	require.Equal(t, srcUser.Name, targetUser.UserName)
	require.NotZero(t, targetUser.ID)
	require.NotZero(t, targetUser.CreatedAt)
	require.NotZero(t, targetUser.UpdatedAt)
}
