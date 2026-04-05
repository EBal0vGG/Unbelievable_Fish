package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type AuctionLister struct {
	db *sql.DB
}

func NewAuctionLister(db *sql.DB) *AuctionLister {
	return &AuctionLister{db: db}
}

func (l *AuctionLister) ListExpired(ctx context.Context, now time.Time, limit int) ([]app.AuctionID, error) {
	if limit <= 0 {
		limit = 100
	}
	const query = `
SELECT auction_id
FROM trading_auctions
WHERE state = $1 AND ends_at <= $2
ORDER BY ends_at
LIMIT $3
`
	dbtx := DBTXFromContext(ctx, l.db)
	rows, err := dbtx.QueryContext(ctx, query, string(auction.StatePublished), now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []app.AuctionID
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, app.AuctionID(id))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
