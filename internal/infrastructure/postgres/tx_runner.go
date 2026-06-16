package postgres

import (
	"context"
	"database/sql"
	"fmt"
)

// contextKey is a private type for context keys in this package.
type contextKey string

const txKey contextKey = "tx"

// TxRunner implements the payments.TxRunner interface using *sql.DB.
type TxRunner struct {
	db *sql.DB
}

// NewTxRunner creates a new TxRunner.
func NewTxRunner(db *sql.DB) *TxRunner {
	return &TxRunner{db: db}
}

// RunInTx executes fn within a database transaction.
// If fn returns an error, the transaction is rolled back.
// The transaction is accessible via TxFromContext within fn.
func (r *TxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op if already committed

	txCtx := context.WithValue(ctx, txKey, tx)
	if err := fn(txCtx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// TxFromContext extracts the *sql.Tx from a context, if present.
// Returns nil if no transaction is in context.
func TxFromContext(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value(txKey).(*sql.Tx)
	return tx
}
