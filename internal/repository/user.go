package repository

import (
	"context"
	"fmt"
	"lazarus/internal/entities"
	"lazarus/internal/storage/database"
	"lazarus/internal/utils"
	"strings"
	"time"
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

func (r *Repo) GetUserByID(ctx context.Context, id int64) (*entities.User, error) {
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
		"email":      usr.Email,
		"user_name":  usr.Name,
		"created_at": time.Now(),
		"updated_at": time.Now(),
		"provider":   "google",
	})
	_, err := r.db.Client().ExecContext(ctx, q, p...)
	return err
}
