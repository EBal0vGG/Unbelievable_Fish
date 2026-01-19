package auction

import "time"

type Event interface {
	isAuctionEvent()
}

type AuctionPublished struct {
	AuctionID string
}

func (AuctionPublished) isAuctionEvent() {}

type BidPlaced struct {
	AuctionID       string
	BidderCompanyID string
	Amount          int64
	PlacedAt        time.Time
}

func (BidPlaced) isAuctionEvent() {}

type AuctionClosed struct {
	AuctionID string
}

func (AuctionClosed) isAuctionEvent() {}

type AuctionWon struct {
	AuctionID       string
	WinnerCompanyID string
	FinalPrice      int64
}

func (AuctionWon) isAuctionEvent() {}

type AuctionCancelled struct {
	AuctionID string
}

func (AuctionCancelled) isAuctionEvent() {}
