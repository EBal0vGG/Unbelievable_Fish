package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type SelectionRepository struct {
	db *sql.DB
}

func NewSelectionRepository(db *sql.DB) *SelectionRepository {
	return &SelectionRepository{db: db}
}

var _ app.WinnerSelectionRepository = (*SelectionRepository)(nil)

func (r *SelectionRepository) Save(ctx context.Context, item *deal.WinnerSelection) error {
	const query = `
INSERT INTO deal_winner_selections (
    auction_id,
    candidates,
    current_index,
    status,
    final_price,
    won_at,
    supplier_id,
    deal_id,
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
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
ON CONFLICT (auction_id) DO UPDATE SET
    candidates = EXCLUDED.candidates,
    current_index = EXCLUDED.current_index,
    status = EXCLUDED.status,
    final_price = EXCLUDED.final_price,
    won_at = EXCLUDED.won_at,
    supplier_id = EXCLUDED.supplier_id,
    deal_id = EXCLUDED.deal_id,
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
	payload, err := json.Marshal(item.Candidates)
	if err != nil {
		return err
	}
	snapshot := item.ProductSnapshot
	dealID := sql.NullString{String: item.DealID, Valid: item.DealID != ""}
	_, err = dbtx.ExecContext(
		ctx,
		query,
		item.AuctionID,
		payload,
		item.CurrentIndex,
		string(item.Status),
		item.FinalPrice,
		item.WonAt,
		item.SupplierID,
		dealID,
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

func (r *SelectionRepository) GetByAuctionID(ctx context.Context, auctionID string) (*deal.WinnerSelection, error) {
	const query = `
SELECT auction_id, candidates, current_index, status, final_price, won_at, supplier_id, deal_id,
       product_id, product_name, product_description, product_category, product_weight,
       product_unit, product_size, product_processing_type, product_volume, product_origin_country
FROM deal_winner_selections
WHERE auction_id = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, auctionID)

	var (
		id, status, supplierID                                                     string
		currentIndex                                                               int
		finalPrice                                                                 int64
		wonAt                                                                      sql.NullTime
		dealID                                                                     sql.NullString
		candidatesRaw                                                              []byte
		productID, productName, productDescription, productCategory, productOrigin string
		productUnit, productSize, productProcessingType                            string
		productWeight, productVolume                                               float64
	)
	if err := row.Scan(
		&id, &candidatesRaw, &currentIndex, &status, &finalPrice, &wonAt, &supplierID, &dealID,
		&productID, &productName, &productDescription, &productCategory, &productWeight,
		&productUnit, &productSize, &productProcessingType, &productVolume, &productOrigin,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, deal.ErrSelectionNotFound
		}
		return nil, err
	}
	if !wonAt.Valid {
		return nil, app.ErrWonAtRequired
	}

	var candidates []string
	if len(candidatesRaw) > 0 {
		if err := json.Unmarshal(candidatesRaw, &candidates); err != nil {
			return nil, err
		}
	}

	return &deal.WinnerSelection{
		AuctionID:    id,
		Candidates:   candidates,
		CurrentIndex: currentIndex,
		Status:       deal.WinnerSelectionStatus(status),
		FinalPrice:   finalPrice,
		WonAt:        wonAt.Time,
		SupplierID:   supplierID,
		DealID:       dealID.String,
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
