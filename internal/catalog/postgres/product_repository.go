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
SELECT product_id, fish_id, seller_company_id, weight, unit, size, processing_type, status
FROM catalog_products
WHERE product_id = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, productID)

	var (
		id              string
		fishID          string
		sellerCompanyID string
		weight          float64
		unit            string
		size            string
		processingType  string
		status          string
	)
	if err := row.Scan(&id, &fishID, &sellerCompanyID, &weight, &unit, &size, &processingType, &status); err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrNotFound
		}
		return nil, err
	}

	return r.productFromRow(id, fishID, sellerCompanyID, weight, unit, size, processingType, status)
}

func (r *ProductRepository) List(ctx context.Context) ([]*catalog.Product, error) {
	const query = `
SELECT product_id, fish_id, seller_company_id, weight, unit, size, processing_type, status
FROM catalog_products
ORDER BY product_id
`
	return r.scanProducts(ctx, query)
}

func (r *ProductRepository) ListBySellerCompany(ctx context.Context, sellerCompanyID string) ([]*catalog.Product, error) {
	const query = `
SELECT product_id, fish_id, seller_company_id, weight, unit, size, processing_type, status
FROM catalog_products
WHERE seller_company_id = $1
ORDER BY product_id
`
	return r.scanProducts(ctx, query, sellerCompanyID)
}

func (r *ProductRepository) scanProducts(ctx context.Context, query string, args ...any) ([]*catalog.Product, error) {
	dbtx := DBTXFromContext(ctx, r.db)
	rows, err := dbtx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*catalog.Product
	for rows.Next() {
		var (
			id              string
			fishID          string
			sellerCompanyID string
			weight          float64
			unit            string
			size            string
			processingType  string
			status          string
		)
		if err := rows.Scan(&id, &fishID, &sellerCompanyID, &weight, &unit, &size, &processingType, &status); err != nil {
			return nil, err
		}
		p, err := r.productFromRow(id, fishID, sellerCompanyID, weight, unit, size, processingType, status)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProductRepository) productFromRow(
	id, fishID, sellerCompanyID string,
	weight float64,
	unit, size, processingType, status string,
) (*catalog.Product, error) {
	product, err := catalog.RestoreProduct(
		id,
		fishID,
		sellerCompanyID,
		weight,
		unit,
		size,
		catalog.ProcessingType(processingType),
		catalog.ProductStatusDraft,
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
    seller_company_id,
    weight,
    unit,
    size,
    processing_type,
    status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (product_id) DO UPDATE SET
    fish_id = EXCLUDED.fish_id,
    seller_company_id = EXCLUDED.seller_company_id,
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
		product.SellerCompanyID(),
		product.Weight(),
		product.Unit(),
		product.Size(),
		string(product.ProcessingType()),
		string(product.Status()),
	)
	return err
}
