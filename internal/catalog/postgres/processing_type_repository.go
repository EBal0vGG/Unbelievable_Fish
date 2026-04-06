package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
)

type ProcessingTypeRepository struct {
	db *sql.DB
}

func NewProcessingTypeRepository(db *sql.DB) *ProcessingTypeRepository {
	return &ProcessingTypeRepository{db: db}
}

var _ app.ProcessingTypeRepository = (*ProcessingTypeRepository)(nil)

func (r *ProcessingTypeRepository) Exists(ctx context.Context, processingType string) (bool, error) {
	const query = `SELECT 1 FROM processing_types WHERE code = $1`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, processingType)
	var exists int
	if err := row.Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
