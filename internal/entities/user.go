package entities

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Email       string    `db:"email" json:"email"`
	DisplayName string    `db:"display_name" json:"display_name"`
	AvatarURL   string    `db:"avatar_url" json:"avatar_url"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

func (u *User) Validate() error {
	if u.ID == uuid.Nil {
		return errors.New("user.id is empty")
	}
	// email can be empty for some providers; don't hard fail
	if len(u.Email) > 320 {
		return errors.New("user.email too long")
	}
	if len(u.DisplayName) > 256 {
		return errors.New("user.display_name too long")
	}
	if len(u.AvatarURL) > 2048 {
		return errors.New("user.avatar_url too long")
	}
	return nil
}
