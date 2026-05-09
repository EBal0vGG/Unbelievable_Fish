// Package postgresctx shares one context key for *sql.Tx across bounded contexts
// (e.g. trading + billing) so repositories participating in the same UnitOfWork see the same transaction.
package postgresctx

import (
	"context"
	"database/sql"
	"errors"
)

// ctxKey is exported type with unexported field so other packages cannot collide on context keys.
type ctxKey struct{ name string }

var txKey = ctxKey{name: "postgresctx.tx"}

// DBTX is the minimal surface repositories need from *sql.DB or *sql.Tx.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Manager runs callbacks inside a database transaction and injects *sql.Tx into context.
type Manager struct {
	db   *sql.DB
	opts *sql.TxOptions
}

// NewManager constructs a Manager. opts may be nil.
func NewManager(db *sql.DB, opts *sql.TxOptions) *Manager {
	return &Manager{db: db, opts: opts}
}

// WithinTx begins a transaction, runs fn with ctx carrying the tx, then commits or rolls back.
func (m *Manager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, m.opts)
	if err != nil {
		return err
	}
	txCtx := context.WithValue(ctx, txKey, tx)
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err := fn(txCtx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}
	return nil
}

// FromContext returns the active transaction if present, otherwise db.
func FromContext(ctx context.Context, db *sql.DB) DBTX {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return db
}

// TxFromContext returns the transaction stored by WithinTx, if any.
func TxFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey).(*sql.Tx)
	return tx, ok
}
