package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
)

type UnitRepository struct {
	db *sql.DB
}

func NewUnitRepository(db *sql.DB) *UnitRepository {
	return &UnitRepository{db: db}
}

var _ app.UnitRepository = (*UnitRepository)(nil)

func (r *UnitRepository) Exists(ctx context.Context, unit string) (bool, error) {
	const query = `SELECT 1 FROM units WHERE code = $1`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, unit)
	var exists int
	if err := row.Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
