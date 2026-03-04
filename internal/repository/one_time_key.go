package repository

import (
	"context"
	"fmt"
	"lazarus/internal/utils"
	"strings"
	"time"

	"github.com/google/uuid"
)

const TableOneTimeKey = "one_time_key"

var (
	oneTimeKeyColumns = []string{
		"key_id",
		"key_val",
		"expires",
	}
	oneTimeKeyColumnsStr = strings.Join(oneTimeKeyColumns, ",")
)

func (r *Repo) GetKey(ctx context.Context, key uuid.UUID) (res string, err error) {
	q := fmt.Sprintf("SELECT key_val FROM %s WHERE key_id = $1 AND expires > NOW()", TableOneTimeKey)
	err = r.db.Client().QueryRowxContext(ctx, q, key).Scan(&res)
	return res, err
}

func (r *Repo) SetKey(ctx context.Context, key string) (uuid.UUID, error) {
	res := uuid.New()
	q, p := utils.GenerateInsertSQL(TableOneTimeKey, map[string]any{
		"key_id":  res.String(),
		"key_val": key,
		"expires": time.Now().Add(10 * time.Minute),
	})
	_, err := r.db.Client().ExecContext(ctx, q, p...)
	return res, err
}
