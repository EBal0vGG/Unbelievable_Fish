package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type AuctionID string

type AuctionRepository interface {
	Load(ctx context.Context, id AuctionID) (*auction.Auction, error)
	Save(ctx context.Context, a *auction.Auction) error
}
