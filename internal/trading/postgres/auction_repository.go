package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type AuctionRepository struct {
	db *sql.DB
}

func NewAuctionRepository(db *sql.DB) *AuctionRepository {
	return &AuctionRepository{db: db}
}

var _ app.AuctionRepository = (*AuctionRepository)(nil)

func (r *AuctionRepository) Save(ctx context.Context, a *auction.Auction) error {
	const query = `
INSERT INTO trading_auctions (
    auction_id,
    lot_id,
    state,
    starts_at,
    ends_at,
    current_price,
    min_bid_step,
    leader_company_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (auction_id) DO UPDATE SET
    lot_id = EXCLUDED.lot_id,
    state = EXCLUDED.state,
    starts_at = EXCLUDED.starts_at,
    ends_at = EXCLUDED.ends_at,
    current_price = EXCLUDED.current_price,
    min_bid_step = EXCLUDED.min_bid_step,
    leader_company_id = EXCLUDED.leader_company_id
`

	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(
		ctx,
		query,
		a.ID,
		a.LotID,
		string(a.State()),
		a.StartsAt(),
		a.EndsAt(),
		a.CurrentPrice(),
		a.MinBidStep(),
		a.LeaderCompanyID(),
	)
	return err
}

func (r *AuctionRepository) Load(ctx context.Context, id app.AuctionID) (*auction.Auction, error) {
	return r.load(ctx, id, false)
}

func (r *AuctionRepository) LoadForUpdate(ctx context.Context, id app.AuctionID) (*auction.Auction, error) {
	return r.load(ctx, id, true)
}

func (r *AuctionRepository) load(ctx context.Context, id app.AuctionID, forUpdate bool) (*auction.Auction, error) {
	query := `
SELECT auction_id, lot_id, state, starts_at, ends_at, current_price, min_bid_step, leader_company_id
FROM trading_auctions
WHERE auction_id = $1
`
	if forUpdate {
		query += " FOR UPDATE"
	}

	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, string(id))

	var (
		auctionID      string
		lotID          string
		state          string
		startsAt       sql.NullTime
		endsAt         sql.NullTime
		currentPrice   int64
		minBidStep     int64
		leaderCompany  string
	)
	if err := row.Scan(&auctionID, &lotID, &state, &startsAt, &endsAt, &currentPrice, &minBidStep, &leaderCompany); err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrNotFound
		}
		return nil, err
	}
	if !startsAt.Valid || !endsAt.Valid {
		return nil, auction.ErrInvalidSchedule
	}
	return auction.RehydrateAuction(
		auctionID,
		lotID,
		auction.State(state),
		startsAt.Time,
		endsAt.Time,
		currentPrice,
		minBidStep,
		leaderCompany,
	)
}
