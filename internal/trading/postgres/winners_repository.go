package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type WinnersRepository struct {
	db *sql.DB
}

func NewWinnersRepository(db *sql.DB) *WinnersRepository {
	return &WinnersRepository{db: db}
}

var _ app.AuctionWinnersRepository = (*WinnersRepository)(nil)

func (r *WinnersRepository) Save(ctx context.Context, auctionID app.AuctionID, winners []app.WinnerRecord) error {
	const deleteQuery = `DELETE FROM trading_auction_winners WHERE auction_id = $1`
	const insertQuery = `
INSERT INTO trading_auction_winners (
    auction_id,
    place,
    company_id,
    amount,
    placed_at
) VALUES ($1, $2, $3, $4, $5)
`

	dbtx := DBTXFromContext(ctx, r.db)
	if _, err := dbtx.ExecContext(ctx, deleteQuery, string(auctionID)); err != nil {
		return err
	}
	for _, winner := range winners {
		if _, err := dbtx.ExecContext(
			ctx,
			insertQuery,
			string(auctionID),
			winner.Place,
			winner.CompanyID,
			winner.Amount,
			winner.PlacedAt,
		); err != nil {
			return err
		}
	}
	return nil
}
