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

func QueryRowsToStruct[T any](ctx context.Context, conn sqlscan.Querier, query string, args ...any) ([]*T, error) {
	rows, err := conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	defer rows.Close()
	res := make([]*T, 0, 100)
	for rows.Next() {
		t := new(T)
		if err = sqlscan.NewRowScanner(rows).Scan(t); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		res = append(res, t)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over rows: %w", err)
	}
	return res, nil
}
