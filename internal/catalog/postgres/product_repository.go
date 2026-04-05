package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

var _ app.ProductRepository = (*ProductRepository)(nil)

func (r *ProductRepository) Get(ctx context.Context, productID string) (*catalog.Product, error) {
	const query = `
SELECT product_id, fish_id, weight, unit, size, processing_type, status
FROM catalog_products
WHERE product_id = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, productID)

	var (
		id             string
		fishID         string
		weight         float64
		unit           string
		size           string
		processingType string
		status         string
	)
	if err := row.Scan(&id, &fishID, &weight, &unit, &size, &processingType, &status); err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrNotFound
		}
		return nil, err
	}

	product, _, err := catalog.NewProduct(
		id,
		fishID,
		weight,
		unit,
		size,
		catalog.ProcessingType(processingType),
	)
	if err != nil {
		return nil, err
	}

	switch catalog.ProductStatus(status) {
	case catalog.ProductStatusDraft, catalog.ProductStatusUnpublished:
		return product, nil
	case catalog.ProductStatusPublished:
		if _, err := product.Publish(); err != nil {
			return nil, err
		}
		return product, nil
	default:
		return nil, fmt.Errorf("unsupported product status: %s", status)
	}
}

func (r *ProductRepository) Save(ctx context.Context, product *catalog.Product) error {
	const query = `
INSERT INTO catalog_products (
    product_id,
    fish_id,
    weight,
    unit,
    size,
    processing_type,
    status
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (product_id) DO UPDATE SET
    fish_id = EXCLUDED.fish_id,
    weight = EXCLUDED.weight,
    unit = EXCLUDED.unit,
    size = EXCLUDED.size,
    processing_type = EXCLUDED.processing_type,
    status = EXCLUDED.status
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(
		ctx,
		query,
		product.ID(),
		product.FishID(),
		product.Weight(),
		product.Unit(),
		product.Size(),
		string(product.ProcessingType()),
		string(product.Status()),
	)
	return err
}
