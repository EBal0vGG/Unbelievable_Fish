package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/postgresctx"
)

// TransactionManager runs use cases inside a single SQL transaction (shared key with billing via postgresctx).
type TransactionManager struct {
	*postgresctx.Manager
}

func NewTransactionManager(db *sql.DB, opts *sql.TxOptions) *TransactionManager {
	return &TransactionManager{Manager: postgresctx.NewManager(db, opts)}
}

func TxFromContext(ctx context.Context) (*sql.Tx, bool) {
	return postgresctx.TxFromContext(ctx)
}

func DBTXFromContext(ctx context.Context, db *sql.DB) postgresctx.DBTX {
	return postgresctx.FromContext(ctx, db)
}
