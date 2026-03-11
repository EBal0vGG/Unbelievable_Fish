package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

func (uc *CreateAuction) Execute(ctx context.Context, meta CommandMeta, lotID string, startsAt, endsAt time.Time) error {
	_ = meta
	id, err := generateAuctionID()
	if err != nil {
		return err
	}
	a, err := auction.NewAuction(id, lotID, startsAt, endsAt)
	if err != nil {
		return err
	}
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
	bidRepo   BidRepository
	pub       EventPublisher
}

func NewPlaceBid(repo AuctionRepository, bidRepo BidRepository, pub EventPublisher) *PlaceBid {
	return &PlaceBid{
		repo:    repo,
		bidRepo: bidRepo,
		pub:     pub,
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
	if err := uc.bidRepo.Save(ctx, id, bid); err != nil {
		return err
	}
	if err := uc.repo.Save(ctx, a); err != nil {
		return err
	}
	return publishEvents(ctx, uc.pub, events)
}

type CloseAuction struct {
	repo      AuctionRepository
	bidRepo   BidRepository
	publisher EventPublisher
}

func NewCloseAuction(repo AuctionRepository, bidRepo BidRepository, publisher EventPublisher) *CloseAuction {
	return &CloseAuction{
		repo:      repo,
		bidRepo:   bidRepo,
		publisher: publisher,
	}
}

func (uc *CloseAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	_ = meta
	a, err := uc.repo.Load(ctx, id)
	if err != nil {
		return err
	}
	bids, err := uc.bidRepo.TopBids(ctx, id)
	if err != nil {
		return err
	}
	events, err := a.Close(bids)
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

func generateAuctionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
