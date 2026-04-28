package app

import (
	"context"
	"time"
)

type AuctionSummary struct {
	AuctionID       AuctionID
	LotID           string
	State           string
	StartsAt        time.Time
	EndsAt          time.Time
	CurrentPrice    int64
	MinBidStep      int64
	LeaderCompanyID string
}

type AuctionReadRepository interface {
	GetByLotID(ctx context.Context, lotID string) (*AuctionSummary, error)
	GetByID(ctx context.Context, id AuctionID) (*AuctionSummary, error)
}

type GetAuctionByLot struct {
	repo AuctionReadRepository
}

func NewGetAuctionByLot(repo AuctionReadRepository) (*GetAuctionByLot, error) {
	if repo == nil {
		return nil, ErrNilAuctionQueryRepository
	}
	return &GetAuctionByLot{repo: repo}, nil
}

func (uc *GetAuctionByLot) Execute(ctx context.Context, lotID string) (*AuctionSummary, error) {
	return uc.repo.GetByLotID(ctx, lotID)
}

type GetAuctionByID struct {
	repo AuctionReadRepository
}

func NewGetAuctionByID(repo AuctionReadRepository) (*GetAuctionByID, error) {
	if repo == nil {
		return nil, ErrNilAuctionQueryRepository
	}
	return &GetAuctionByID{repo: repo}, nil
}

func (uc *GetAuctionByID) Execute(ctx context.Context, id AuctionID) (*AuctionSummary, error) {
	return uc.repo.GetByID(ctx, id)
}
