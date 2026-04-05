package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type UnitOfWork struct {
	db   *sql.DB
	tx   *TransactionManager
	auctions *AuctionRepository
	bids     *BidRepository
	outbox   *OutboxRepository
	winners  *WinnersRepository
}

func NewUnitOfWork(db *sql.DB) *UnitOfWork {
	return &UnitOfWork{
		db:       db,
		tx:       NewTransactionManager(db, nil),
		auctions: NewAuctionRepository(db),
		bids:     NewBidRepository(db),
		outbox:   NewOutboxRepository(db),
		winners:  NewWinnersRepository(db),
	}
}

func (u *UnitOfWork) Do(ctx context.Context, fn func(app.Tx) error) error {
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		return fn(&tx{
			auctions: u.auctions,
			bids:     u.bids,
			outbox:   u.outbox,
			winners:  u.winners,
		})
	})
}

type tx struct {
	auctions *AuctionRepository
	bids     *BidRepository
	outbox   *OutboxRepository
	winners  *WinnersRepository
}

func (t *tx) Auctions() app.AuctionRepository { return t.auctions }
func (t *tx) Bids() app.BidRepository         { return t.bids }
func (t *tx) Outbox() app.OutboxRepository    { return t.outbox }
func (t *tx) Winners() app.AuctionWinnersRepository { return t.winners }
