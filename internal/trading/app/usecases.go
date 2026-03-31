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

func NewCreateAuction(uow UnitOfWork, factory AuctionIDFactory) *CreateAuction {
	if uow == nil {
		panic("nil UnitOfWork")
	}
	if factory == nil {
		panic("nil AuctionIDFactory")
	}
	return &CreateAuction{
		uow:     uow,
		factory: factory,
	}
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

func NewPublishAuction(uow UnitOfWork) *PublishAuction {
	if uow == nil {
		panic("nil UnitOfWork")
	}
	return &PublishAuction{
		uow: uow,
	}
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

func NewPlaceBid(uow UnitOfWork) *PlaceBid {
	if uow == nil {
		panic("nil UnitOfWork")
	}
	return &PlaceBid{
		uow: uow,
	}
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

func NewCloseAuction(uow UnitOfWork) *CloseAuction {
	if uow == nil {
		panic("nil UnitOfWork")
	}
	return &CloseAuction{
		uow: uow,
	}
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

func NewCancelAuction(uow UnitOfWork) *CancelAuction {
	if uow == nil {
		panic("nil UnitOfWork")
	}
	return &CancelAuction{
		uow: uow,
	}
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
