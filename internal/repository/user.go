package repository

import (
	"context"
	"fmt"
	"lazarus/internal/entities"
	"lazarus/internal/storage/database"
	"lazarus/internal/utils"
	"strings"
	"time"

	"github.com/google/uuid"
)

const TableUser = "users"

var (
	userColumns = []string{
		"u_id",
		"email",
		"user_name",
		"created_at",
		"updated_at",
	}
	userColumnsStr = strings.Join(userColumns, ",")
)

func (r *Repo) GetUserByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	q := fmt.Sprintf("SELECT %s FROM %s WHERE u_id = $1", userColumnsStr, TableUser)
	res, err := database.QueryRowToStruct[entities.User](ctx, r.db.Client(), q, id)
	if err != nil {
		if database.NoRowsInResultSet(err) {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (r *Repo) GetUserByMail(ctx context.Context, email string) (*entities.User, error) {
	q := fmt.Sprintf("SELECT %s FROM %s WHERE email = $1", userColumnsStr, TableUser)
	res, err := database.QueryRowToStruct[entities.User](ctx, r.db.Client(), q, email)
	if err != nil {
		if database.NoRowsInResultSet(err) {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (r *Repo) AddGoogleUser(ctx context.Context, usr *entities.GoogleUser) error {
	q, p := utils.GenerateInsertSQL(TableUser, map[string]any{
		"u_id":        uuid.NewString(),
		"email":       usr.Email,
		"user_name":   usr.Name,
		"user_locale": "en",
		"created_at":  time.Now(),
		"updated_at":  time.Now(),
		"provider":    "google",
	})
	_, err := r.db.Client().ExecContext(ctx, q, p...)
	return err
}

func (r *Repo) GetAllUsers(ctx context.Context) ([]*entities.User, error) {
	q := fmt.Sprintf("SELECT %s FROM %s", userColumnsStr, TableUser)
	return database.QueryRowsToStruct[entities.User](ctx, r.db.Client(), q)
}
