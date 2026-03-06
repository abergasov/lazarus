package seed

import (
	"context"
	"lazarus/internal/entities"
	testhelpers "lazarus/internal/test_helpers"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type UserBuilder struct {
	user *entities.GoogleUser
}

func NewUserBuilder() *UserBuilder {
	return &UserBuilder{
		user: &entities.GoogleUser{
			Email: uuid.NewString()[:8] + "@example.com",
			Name:  uuid.NewString(),
		},
	}
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.user.Email = email
	return b
}

func (b *UserBuilder) WithName(name string) *UserBuilder {
	b.user.Name = name
	return b
}

func (b *UserBuilder) Build() *entities.GoogleUser {
	return b.user
}

func (b *UserBuilder) PopulateTests(t *testing.T, cnt *testhelpers.TestContainer) *entities.User {
	ctx, cancel := context.WithTimeout(cnt.Ctx, 10*time.Second)
	defer cancel()
	require.NoError(t, cnt.Repo.AddGoogleUser(ctx, b.Build()))
	usr, err := cnt.Repo.GetUserByMail(ctx, b.user.Email)
	require.NoError(t, err)
	require.NotNil(t, usr)
	return usr
}
