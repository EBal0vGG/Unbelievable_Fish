package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
)

type FishRepository struct {
	db *sql.DB
}

func NewFishRepository(db *sql.DB) *FishRepository {
	return &FishRepository{db: db}
}

var _ app.FishRepository = (*FishRepository)(nil)

func (r *FishRepository) Get(ctx context.Context, fishID string) (*catalog.Fish, error) {
	const query = `
SELECT id, name, description
FROM fish
WHERE id = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, fishID)

	var id, name, description string
	if err := row.Scan(&id, &name, &description); err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrNotFound
		}
		return nil, err
	}
	return catalog.NewFish(id, name, description)
}

func (r *FishRepository) Exists(ctx context.Context, fishID string) (bool, error) {
	const query = `SELECT 1 FROM fish WHERE id = $1`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, fishID)
	var exists int
	if err := row.Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *FishRepository) List(ctx context.Context) ([]*catalog.Fish, error) {
	const query = `
SELECT id, name, description
FROM fish
ORDER BY name, id
`
	dbtx := DBTXFromContext(ctx, r.db)
	rows, err := dbtx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*catalog.Fish
	for rows.Next() {
		var id, name, description string
		if err := rows.Scan(&id, &name, &description); err != nil {
			return nil, err
		}
		fish, err := catalog.NewFish(id, name, description)
		if err != nil {
			return nil, err
		}
		out = append(out, fish)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *FishRepository) Save(ctx context.Context, fish *catalog.Fish) error {
	const query = `
INSERT INTO fish (id, name, description)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(
		ctx,
		query,
		fish.ID(),
		fish.Name(),
		fish.Description(),
	)
	return err
}
