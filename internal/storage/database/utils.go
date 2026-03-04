package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/georgysavva/scany/v2/sqlscan"
)

func NoRowsInResultSet(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no rows in result set")
}

func QueryRowToStruct[T any](ctx context.Context, conn sqlscan.Querier, query string, args ...any) (*T, error) {
	var t T
	if err := sqlscan.Get(ctx, conn, &t, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get row: %w", err)
	}
	return &t, nil
}
