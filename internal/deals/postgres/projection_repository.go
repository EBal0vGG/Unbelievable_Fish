package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type ProjectionRepository struct {
	db *sql.DB
}

func NewProjectionRepository(db *sql.DB) *ProjectionRepository {
	return &ProjectionRepository{db: db}
}

var _ app.ProjectionRepository = (*ProjectionRepository)(nil)

func (r *ProjectionRepository) Save(ctx context.Context, item *deal.DealProjection) error {
	const query = `
INSERT INTO deal_projections (
    auction_id,
    supplier_id,
    start_price,
    published_at,
    status,
    product_id,
    product_name,
    product_description,
    product_category,
    product_weight,
    product_unit,
    product_size,
    product_processing_type,
    product_volume,
    product_origin_country
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
ON CONFLICT (auction_id) DO UPDATE SET
    supplier_id = EXCLUDED.supplier_id,
    start_price = EXCLUDED.start_price,
    published_at = EXCLUDED.published_at,
    status = EXCLUDED.status,
    product_id = EXCLUDED.product_id,
    product_name = EXCLUDED.product_name,
    product_description = EXCLUDED.product_description,
    product_category = EXCLUDED.product_category,
    product_weight = EXCLUDED.product_weight,
    product_unit = EXCLUDED.product_unit,
    product_size = EXCLUDED.product_size,
    product_processing_type = EXCLUDED.product_processing_type,
    product_volume = EXCLUDED.product_volume,
    product_origin_country = EXCLUDED.product_origin_country
`
	dbtx := DBTXFromContext(ctx, r.db)
	snapshot := item.ProductSnapshot
	_, err := dbtx.ExecContext(
		ctx,
		query,
		item.AuctionID,
		item.SupplierID,
		item.StartPrice,
		item.PublishedAt,
		string(item.Status),
		snapshot.ProductID,
		snapshot.Name,
		snapshot.Description,
		snapshot.Category,
		snapshot.Weight,
		snapshot.Unit,
		snapshot.Size,
		snapshot.ProcessingType,
		snapshot.Volume,
		snapshot.OriginCountry,
	)
	return err
}

func (r *ProjectionRepository) GetByAuctionID(ctx context.Context, auctionID string) (*deal.DealProjection, error) {
	const query = `
SELECT auction_id, supplier_id, start_price, published_at, status,
       product_id, product_name, product_description, product_category,
       product_weight, product_unit, product_size, product_processing_type, product_volume, product_origin_country
FROM deal_projections
WHERE auction_id = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, auctionID)

	var (
		id, supplier, status                                                       string
		startPrice                                                                 int64
		publishedAt                                                                sql.NullTime
		productID, productName, productDescription, productCategory, productOrigin string
		productUnit, productSize, productProcessingType                            string
		productWeight, productVolume                                               float64
	)
	if err := row.Scan(
		&id, &supplier, &startPrice, &publishedAt, &status,
		&productID, &productName, &productDescription, &productCategory,
		&productWeight, &productUnit, &productSize, &productProcessingType, &productVolume, &productOrigin,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, deal.ErrProjectionNotFound
		}
		return nil, err
	}
	if !publishedAt.Valid {
		return nil, app.ErrPublishedAtRequired
	}

	return &deal.DealProjection{
		AuctionID:   id,
		SupplierID:  supplier,
		StartPrice:  startPrice,
		PublishedAt: publishedAt.Time,
		Status:      deal.ProjectionStatus(status),
		ProductSnapshot: deal.ProductSnapshot{
			ProductID:      productID,
			Name:           productName,
			Description:    productDescription,
			Category:       productCategory,
			Weight:         productWeight,
			Unit:           productUnit,
			Size:           productSize,
			ProcessingType: productProcessingType,
			Volume:         productVolume,
			OriginCountry:  productOrigin,
		},
	}, nil
}
