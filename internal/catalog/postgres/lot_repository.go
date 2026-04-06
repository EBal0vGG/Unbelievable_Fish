package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
)

type LotRepository struct {
	db *sql.DB
}

func NewLotRepository(db *sql.DB) *LotRepository {
	return &LotRepository{db: db}
}

var _ app.LotRepository = (*LotRepository)(nil)

func (r *LotRepository) Get(ctx context.Context, lotID string) (*catalog.Lot, error) {
	const query = `
SELECT lot_id, product_id, auction_id, seller_company_id, photo, quantity, start_price, cur_price, final_price, status, auction_starts_at, auction_duration_minutes
FROM catalog_lots
WHERE lot_id = $1
`

	return r.getOne(ctx, query, lotID)
}

func (r *LotRepository) GetByAuctionID(ctx context.Context, auctionID string) (*catalog.Lot, error) {
	const query = `
SELECT lot_id, product_id, auction_id, seller_company_id, photo, quantity, start_price, cur_price, final_price, status, auction_starts_at, auction_duration_minutes
FROM catalog_lots
WHERE auction_id = $1
`

	return r.getOne(ctx, query, auctionID)
}

func (r *LotRepository) Save(ctx context.Context, lot *catalog.Lot) error {
	const query = `
INSERT INTO catalog_lots (
    lot_id,
    product_id,
    auction_id,
    seller_company_id,
    photo,
    quantity,
    start_price,
    cur_price,
    final_price,
    status,
    auction_starts_at,
    auction_duration_minutes
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (lot_id) DO UPDATE SET
    product_id = EXCLUDED.product_id,
    auction_id = EXCLUDED.auction_id,
    seller_company_id = EXCLUDED.seller_company_id,
    photo = EXCLUDED.photo,
    quantity = EXCLUDED.quantity,
    start_price = EXCLUDED.start_price,
    cur_price = EXCLUDED.cur_price,
    final_price = EXCLUDED.final_price,
    status = EXCLUDED.status,
    auction_starts_at = EXCLUDED.auction_starts_at,
    auction_duration_minutes = EXCLUDED.auction_duration_minutes
`

	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(
		ctx,
		query,
		lot.ID(),
		lot.ProductID(),
		nullIfBlank(lot.AuctionID()),
		lot.SellerCompanyID(),
		nullIfBlank(lot.Photo()),
		lot.Quantity(),
		lot.StartPrice(),
		lot.CurPrice(),
		lot.FinalPrice(),
		string(lot.Status()),
		lot.AuctionStartsAt(),
		int64(lot.AuctionSchedule().Duration().Minutes()),
	)
	return err
}

func (r *LotRepository) getOne(ctx context.Context, query string, arg string) (*catalog.Lot, error) {
	dbtx := DBTXFromContext(ctx, r.db)

	var row lotRow
	err := dbtx.QueryRowContext(ctx, query, arg).Scan(
		&row.LotID,
		&row.ProductID,
		&row.AuctionID,
		&row.SellerCompanyID,
		&row.Photo,
		&row.Quantity,
		&row.StartPrice,
		&row.CurPrice,
		&row.FinalPrice,
		&row.Status,
		&row.AuctionStartsAt,
		&row.AuctionDurationMinutes,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrNotFound
		}
		return nil, err
	}

	return row.toAggregate()
}

type lotRow struct {
	LotID           string
	ProductID       string
	AuctionID       sql.NullString
	SellerCompanyID string
	Photo           sql.NullString
	Quantity        float64
	StartPrice      int64
	CurPrice        int64
	FinalPrice      int64
	Status          string
	AuctionStartsAt sql.NullTime
	AuctionDurationMinutes int64
}

func (r lotRow) toAggregate() (*catalog.Lot, error) {
	if !r.AuctionStartsAt.Valid || r.AuctionDurationMinutes <= 0 {
		return nil, catalog.ErrInvalidSchedule
	}

	lot, _, err := catalog.NewLot(
		r.LotID,
		r.ProductID,
		r.SellerCompanyID,
		r.Photo.String,
		r.Quantity,
		r.StartPrice,
		catalog.NewAuctionScheduleAt(r.AuctionStartsAt.Time, time.Duration(r.AuctionDurationMinutes)*time.Minute),
	)
	if err != nil {
		return nil, err
	}

	if r.AuctionID.Valid && r.AuctionID.String != "" {
		if _, err := lot.AssignAuctionID(r.AuctionID.String); err != nil {
			return nil, err
		}
	}

	switch catalog.LotStatus(r.Status) {
	case catalog.LotStatusDraft:
		return lot, nil
	case catalog.LotStatusPublished:
		return rehydratePublishedLot(lot, r.CurPrice)
	case catalog.LotStatusCancelled:
		published, err := rehydratePublishedLot(lot, r.CurPrice)
		if err != nil {
			return nil, err
		}
		if _, err := published.Unpublish(); err != nil {
			return nil, err
		}
		return published, nil
	case catalog.LotStatusClosed:
		published, err := rehydratePublishedLot(lot, r.CurPrice)
		if err != nil {
			return nil, err
		}
		if _, err := published.Close(r.FinalPrice); err != nil {
			return nil, err
		}
		return published, nil
	default:
		return nil, fmt.Errorf("unsupported lot status: %s", r.Status)
	}
}

func rehydratePublishedLot(lot *catalog.Lot, curPrice int64) (*catalog.Lot, error) {
	if _, err := lot.Publish(true, catalog.ProductSnapshot{}); err != nil {
		return nil, err
	}
	if curPrice != lot.StartPrice() {
		if _, err := lot.UpdateCurrentPrice(curPrice); err != nil {
			return nil, err
		}
	}
	return lot, nil
}

func nullIfBlank(value string) any {
	if value == "" {
		return nil
	}
	return value
}
