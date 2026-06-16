package postgres

import (
	"context"
	"database/sql"
)

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func executorForContext(ctx context.Context, db *sql.DB) sqlExecutor {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return db
}
