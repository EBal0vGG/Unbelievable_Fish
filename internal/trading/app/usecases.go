package app

import (
	"context"
	"errors"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type CreateAuction struct {
	uow     UnitOfWork
	factory AuctionIDFactory
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

func (uc *CreateAuction) Execute(
	ctx context.Context,
	meta CommandMeta,
	lotID string,
	startsAt, endsAt time.Time,
	startPrice int64,
	minBidStep int64,
) (AuctionID, error) {
	if minBidStep <= 0 {
		minBidStep = 1
	}
	id, err := uc.factory.NewID()
	if err != nil {
		return "", err
	}
	err = uc.uow.Do(ctx, func(tx Tx) error {
		if _, err := tx.Auctions().Load(ctx, id); err == nil {
			return nil
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		a, err := auction.NewAuctionWithPricing(string(id), lotID, startsAt, endsAt, startPrice, minBidStep)
		if err != nil {
			return err
		}
		return tx.Auctions().Save(ctx, a)
	})
	if err != nil {
		return "", err
	}
	return id, nil
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
		a, err := tx.Auctions().LoadForUpdate(ctx, id)
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
		return tx.Outbox().Add(WithCommandMeta(ctx, meta), events)
	})
}

type PlaceBid struct {
	uow      UnitOfWork
	deposits DepositService
}

func NewPlaceBid(uow UnitOfWork, deposits DepositService) (*PlaceBid, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	if deposits == nil {
		return nil, ErrNilDepositService
	}
	return &PlaceBid{
		uow:      uow,
		deposits: deposits,
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
		a, err := tx.Auctions().LoadForUpdate(ctx, id)
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
		if err := uc.deposits.ReserveAuctionDeposit(ctx, meta.CompanyID, string(id), a.StartPrice()); err != nil {
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
		return tx.Outbox().Add(WithCommandMeta(ctx, meta), events)
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

func allowPrivilegedEarlyAuctionClose(meta CommandMeta) bool {
	switch effectiveActorKind(meta) {
	case ActorKindPlatformAdmin, ActorKindSystem:
		return true
	default:
		return false
	}
}

func (uc *CloseAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	return uc.uow.Do(ctx, func(tx Tx) error {
		a, err := tx.Auctions().LoadForUpdate(ctx, id)
		if err != nil {
			return err
		}
		bids, err := tx.Bids().TopBids(ctx, id)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if a.State() == auction.StatePublished && len(bids) > 0 && now.Before(a.EndsAt()) {
			if !allowPrivilegedEarlyAuctionClose(meta) {
				return ErrCloseForbiddenBeforeEndWithBids
			}
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
		return tx.Outbox().Add(WithCommandMeta(ctx, meta), events)
	})
}

type SellerCancelAuction struct {
	uow UnitOfWork
}

func NewSellerCancelAuction(uow UnitOfWork) (*SellerCancelAuction, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &SellerCancelAuction{uow: uow}, nil
}

func (uc *SellerCancelAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	return uc.uow.Do(ctx, func(tx Tx) error {
		a, err := tx.Auctions().LoadForUpdate(ctx, id)
		if err != nil {
			return err
		}
		events, err := a.CancelAsSellerOrDraft()
		if err != nil {
			return err
		}
		if err := tx.Auctions().Save(ctx, a); err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		return tx.Outbox().Add(WithCommandMeta(ctx, meta), events)
	})
}

type AdminCancelAuction struct {
	uow UnitOfWork
}

func NewAdminCancelAuction(uow UnitOfWork) (*AdminCancelAuction, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &AdminCancelAuction{uow: uow}, nil
}

func (uc *AdminCancelAuction) Execute(ctx context.Context, meta CommandMeta, id AuctionID) error {
	return uc.uow.Do(ctx, func(tx Tx) error {
		a, err := tx.Auctions().LoadForUpdate(ctx, id)
		if err != nil {
			return err
		}
		events, err := a.CancelAsAdmin()
		if err != nil {
			return err
		}
		if err := tx.Auctions().Save(ctx, a); err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		return tx.Outbox().Add(WithCommandMeta(ctx, meta), events)
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
