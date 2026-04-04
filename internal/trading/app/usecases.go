package app

import (
	"context"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type CreateAuction struct {
	uow       UnitOfWork
	factory   AuctionIDFactory
}

func NewCreateAuction(uow UnitOfWork, factory AuctionIDFactory) (*CreateAuction, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	if factory == nil {
		return nil, ErrNilAuctionIDFactory
	}
	return &CreateAuction{
		uow:     uow,
		factory: factory,
	}, nil
}

func (uc *CreateAuction) Execute(ctx context.Context, meta CommandMeta, lotID string, startsAt, endsAt time.Time) error {
	id, err := uc.factory.NewID()
	if err != nil {
		return err
	}
	a, err := auction.NewAuction(string(id), lotID, startsAt, endsAt)
	if err != nil {
		return err
	}
	return uc.uow.Do(ctx, func(tx Tx) error {
		return tx.Auctions().Save(ctx, a)
	})
}

type PublishAuction struct {
	uow UnitOfWork
}

func NewPublishAuction(uow UnitOfWork) (*PublishAuction, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &PublishAuction{
		uow: uow,
	}, nil
}

func (uc *PublishAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	return uc.uow.Do(ctx, func(tx Tx) error {
		a, err := tx.Auctions().Load(ctx, id)
		if err != nil {
			return err
		}
		events, err := a.Publish()
		if err != nil {
			return err
		}
		if err := tx.Auctions().Save(ctx, a); err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		return tx.Outbox().Save(ctx, NewEnvelope(meta, events))
	})
}

type PlaceBid struct {
	uow UnitOfWork
}

func NewPlaceBid(uow UnitOfWork) (*PlaceBid, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &PlaceBid{
		uow: uow,
	}, nil
}

func (uc *PlaceBid) Execute(
	ctx context.Context,
	meta CommandMeta,
	id AuctionID,
	amount int64,
	placedAt time.Time,
) error {
	return uc.uow.Do(ctx, func(tx Tx) error {
		a, err := tx.Auctions().Load(ctx, id)
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
		if err := tx.Bids().Save(ctx, id, bid); err != nil {
			return err
		}
		if err := tx.Auctions().Save(ctx, a); err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		return tx.Outbox().Save(ctx, NewEnvelope(meta, events))
	})
}

type CloseAuction struct {
	uow UnitOfWork
}

func NewCloseAuction(uow UnitOfWork) (*CloseAuction, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &CloseAuction{
		uow: uow,
	}, nil
}

func (uc *CloseAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	return uc.uow.Do(ctx, func(tx Tx) error {
		a, err := tx.Auctions().Load(ctx, id)
		if err != nil {
			return err
		}
		bids, err := tx.Bids().TopBids(ctx, id)
		if err != nil {
			return err
		}
		events, err := a.Close(bids)
		if err != nil {
			return err
		}
		if err := tx.Winners().Save(ctx, id, winnerRecordsFromBids(bids, 3)); err != nil {
			return err
		}
		if err := tx.Auctions().Save(ctx, a); err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		return tx.Outbox().Save(ctx, NewEnvelope(meta, events))
	})
}

type CancelAuction struct {
	uow UnitOfWork
}

func NewCancelAuction(uow UnitOfWork) (*CancelAuction, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &CancelAuction{
		uow: uow,
	}, nil
}

func (uc *CancelAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	return uc.uow.Do(ctx, func(tx Tx) error {
		a, err := tx.Auctions().Load(ctx, id)
		if err != nil {
			return err
		}
		events, err := a.Cancel()
		if err != nil {
			return err
		}
		if err := tx.Auctions().Save(ctx, a); err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		return tx.Outbox().Save(ctx, NewEnvelope(meta, events))
	})
}

func winnerRecordsFromBids(bids []auction.Bid, limit int) []WinnerRecord {
	if len(bids) == 0 || limit <= 0 {
		return nil
	}
	if limit > len(bids) {
		limit = len(bids)
	}
	out := make([]WinnerRecord, 0, limit)
	for i := 0; i < limit; i++ {
		bid := bids[i]
		out = append(out, WinnerRecord{
			Place:     i + 1,
			CompanyID: bid.BidderCompanyID(),
			Amount:    bid.Amount(),
			PlacedAt:  bid.PlacedAt(),
		})
	}
	return out
}
