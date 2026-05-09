package app

import "time"

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type AuctionID string

type AuctionRepository interface {
	Load(ctx context.Context, id AuctionID) (*auction.Auction, error)
	LoadForUpdate(ctx context.Context, id AuctionID) (*auction.Auction, error)
	Save(ctx context.Context, a *auction.Auction) error
}

type BidRepository interface {
	Save(ctx context.Context, auctionID AuctionID, bid auction.Bid) error
	TopBids(ctx context.Context, auctionID AuctionID) ([]auction.Bid, error)
}

type OutboxRepository interface {
	Add(ctx context.Context, events []auction.Event) error
}

type WinnerRecord struct {
	Place     int
	CompanyID string
	Amount    int64
	PlacedAt  time.Time
}

type AuctionWinnersRepository interface {
	Save(ctx context.Context, auctionID AuctionID, winners []WinnerRecord) error
}

type Tx interface {
	Auctions() AuctionRepository
	Bids() BidRepository
	Outbox() OutboxRepository
	Winners() AuctionWinnersRepository
}

type UnitOfWork interface {
	Do(ctx context.Context, fn func(Tx) error) error
}

// DepositService reserves auction deposit funds in the same DB transaction as PlaceBid (implemented via billing in composition root).
type DepositService interface {
	ReserveAuctionDeposit(ctx context.Context, companyID, auctionID string, startPrice int64) error
}

// NoopDepositService is a DepositService that does nothing (tests or callers without billing).
type NoopDepositService struct{}

func (NoopDepositService) ReserveAuctionDeposit(ctx context.Context, companyID, auctionID string, startPrice int64) error {
	_ = ctx
	_ = companyID
	_ = auctionID
	_ = startPrice
	return nil
}
