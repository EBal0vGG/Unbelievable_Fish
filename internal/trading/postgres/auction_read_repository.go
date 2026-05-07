package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type AuctionReadRepository struct {
	db *sql.DB
}

func NewAuctionReadRepository(db *sql.DB) *AuctionReadRepository {
	return &AuctionReadRepository{db: db}
}

var _ app.AuctionReadRepository = (*AuctionReadRepository)(nil)

func (r *AuctionReadRepository) List(ctx context.Context) ([]*app.AuctionSummary, error) {
	const query = `
SELECT auction_id, lot_id, state, starts_at, ends_at, start_price, current_price, min_bid_step, leader_company_id
FROM trading_auctions
WHERE state <> 'DRAFT'
ORDER BY starts_at DESC, auction_id DESC
`
	dbtx := DBTXFromContext(ctx, r.db)
	rows, err := dbtx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*app.AuctionSummary
	for rows.Next() {
		var item app.AuctionSummary
		if err := rows.Scan(&item.AuctionID, &item.LotID, &item.State, &item.StartsAt, &item.EndsAt, &item.StartPrice, &item.CurrentPrice, &item.MinBidStep, &item.LeaderCompanyID); err != nil {
			return nil, err
		}
		out = append(out, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *AuctionReadRepository) GetByLotID(ctx context.Context, lotID string) (*app.AuctionSummary, error) {
	const query = `
SELECT auction_id, lot_id, state, starts_at, ends_at, start_price, current_price, min_bid_step, leader_company_id
FROM trading_auctions
WHERE lot_id = $1
ORDER BY starts_at DESC
LIMIT 1
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, lotID)
	var out app.AuctionSummary
	if err := row.Scan(&out.AuctionID, &out.LotID, &out.State, &out.StartsAt, &out.EndsAt, &out.StartPrice, &out.CurrentPrice, &out.MinBidStep, &out.LeaderCompanyID); err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *AuctionReadRepository) GetByID(ctx context.Context, id app.AuctionID) (*app.AuctionSummary, error) {
	const query = `
SELECT auction_id, lot_id, state, starts_at, ends_at, start_price, current_price, min_bid_step, leader_company_id
FROM trading_auctions
WHERE auction_id = $1
LIMIT 1
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, id)
	var out app.AuctionSummary
	if err := row.Scan(&out.AuctionID, &out.LotID, &out.State, &out.StartsAt, &out.EndsAt, &out.StartPrice, &out.CurrentPrice, &out.MinBidStep, &out.LeaderCompanyID); err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}
