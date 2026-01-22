package app

import (
	"context"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type CreateAuction struct {
	repo      AuctionRepository
	publisher EventPublisher
}

func NewCreateAuction(repo AuctionRepository, publisher EventPublisher) *CreateAuction {
	return &CreateAuction{
		repo:      repo,
		publisher: publisher,
	}
}

func (uc *CreateAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	_ = meta
	a := auction.NewAuction(string(id))
	if err := uc.repo.Save(ctx, a); err != nil {
		return err
	}
	return publishEvents(ctx, uc.publisher, nil)
}

type PublishAuction struct {
	repo      AuctionRepository
	publisher EventPublisher
}

func NewPublishAuction(repo AuctionRepository, publisher EventPublisher) *PublishAuction {
	return &PublishAuction{
		repo:      repo,
		publisher: publisher,
	}
}

func (uc *PublishAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	_ = meta
	a, err := uc.repo.Load(ctx, id)
	if err != nil {
		return err
	}
	events, err := a.Publish()
	if err != nil {
		return err
	}
	if err := uc.repo.Save(ctx, a); err != nil {
		return err
	}
	return publishEvents(ctx, uc.publisher, events)
}

type PlaceBid struct {
	repo      AuctionRepository
	publisher EventPublisher
}

func NewPlaceBid(repo AuctionRepository, publisher EventPublisher) *PlaceBid {
	return &PlaceBid{
		repo:      repo,
		publisher: publisher,
	}
}

func (uc *PlaceBid) Execute(
	ctx context.Context,
	meta CommandMeta,
	id AuctionID,
	amount int64,
	placedAt time.Time,
) error {
	a, err := uc.repo.Load(ctx, id)
	if err != nil {
		return err
	}
	bid, err := auction.NewBid(meta.CompanyID, amount, placedAt)
	if err != nil {
		return err
	}
	events, err := a.PlaceBid(bid)
	if err != nil {
		return err
	}
	if err := uc.repo.Save(ctx, a); err != nil {
		return err
	}
	return publishEvents(ctx, uc.publisher, events)
}

type CloseAuction struct {
	repo      AuctionRepository
	publisher EventPublisher
}

func NewCloseAuction(repo AuctionRepository, publisher EventPublisher) *CloseAuction {
	return &CloseAuction{
		repo:      repo,
		publisher: publisher,
	}
}

func (uc *CloseAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	_ = meta
	a, err := uc.repo.Load(ctx, id)
	if err != nil {
		return err
	}
	events, err := a.Close()
	if err != nil {
		return err
	}
	if err := uc.repo.Save(ctx, a); err != nil {
		return err
	}
	return publishEvents(ctx, uc.publisher, events)
}

type CancelAuction struct {
	repo      AuctionRepository
	publisher EventPublisher
}

func NewCancelAuction(repo AuctionRepository, publisher EventPublisher) *CancelAuction {
	return &CancelAuction{
		repo:      repo,
		publisher: publisher,
	}
}

func (uc *CancelAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	_ = meta
	a, err := uc.repo.Load(ctx, id)
	if err != nil {
		return err
	}
	events, err := a.Cancel()
	if err != nil {
		return err
	}
	if err := uc.repo.Save(ctx, a); err != nil {
		return err
	}
	return publishEvents(ctx, uc.publisher, events)
}

func publishEvents(ctx context.Context, publisher EventPublisher, events []auction.Event) error {
	if publisher == nil || len(events) == 0 {
		return nil
	}
	return publisher.Publish(ctx, events)
}
