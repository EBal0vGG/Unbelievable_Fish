package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type BidRepository struct {
	db *sql.DB
}

func NewBidRepository(db *sql.DB) *BidRepository {
	return &BidRepository{db: db}
}

var _ app.BidRepository = (*BidRepository)(nil)

func (r *BidRepository) Save(ctx context.Context, auctionID app.AuctionID, bid auction.Bid) error {
	const query = `
INSERT INTO trading_bids (
    auction_id,
    bidder_company_id,
    amount,
    placed_at
) VALUES ($1, $2, $3, $4)
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(
		ctx,
		query,
		string(auctionID),
		bid.BidderCompanyID(),
		bid.Amount(),
		bid.PlacedAt(),
	)
	return err
}

func (r *BidRepository) TopBids(ctx context.Context, auctionID app.AuctionID) ([]auction.Bid, error) {
	const query = `
SELECT bidder_company_id, amount, placed_at
FROM trading_bids
WHERE auction_id = $1
ORDER BY amount DESC, placed_at ASC
`
	dbtx := DBTXFromContext(ctx, r.db)
	rows, err := dbtx.QueryContext(ctx, query, string(auctionID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bids []auction.Bid
	for rows.Next() {
		var bidder string
		var amount int64
		var placedAt sql.NullTime
		if err := rows.Scan(&bidder, &amount, &placedAt); err != nil {
			return nil, err
		}
		if !placedAt.Valid {
			continue
		}
		bid, err := auction.NewBid(bidder, amount, placedAt.Time)
		if err != nil {
			return nil, err
		}
		bids = append(bids, bid)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bids, nil
}
