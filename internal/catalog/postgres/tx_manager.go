package postgres

import (
	"context"
	"database/sql"
	"errors"
)

type txContextKey struct{}

type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// TransactionManager реализует app.TransactionManager поверх database/sql.
// Он открывает реальную транзакцию БД и прокидывает *sql.Tx в context для репозиториев/outbox.
type TransactionManager struct {
	db   *sql.DB
	opts *sql.TxOptions
}

func NewTransactionManager(db *sql.DB, opts *sql.TxOptions) *TransactionManager {
	return &TransactionManager{
		db:   db,
		opts: opts,
	}
}

func (m *TransactionManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, m.opts)
	if err != nil {
		return err
	}

	txCtx := context.WithValue(ctx, txContextKey{}, tx)

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

func TxFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(*sql.Tx)
	return tx, ok
}

func DBTXFromContext(ctx context.Context, db *sql.DB) DBTX {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return db
}
