package entities

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
}

type User struct {
	ID        uuid.UUID `db:"u_id" json:"id"`
	Email     string    `db:"email" json:"email"`
	UserName  string    `db:"user_name" json:"user_name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type UserJWT struct {
	UserID uuid.UUID `json:"id"`
	jwt.RegisteredClaims
}

func (u *UserJWT) GetUserID() uuid.UUID {
	return u.UserID
}
