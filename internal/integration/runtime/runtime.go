package runtime

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalog "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	deal "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/eventbus/inmemory"
	outbox "github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox"
	outboxpg "github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox/postgres"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/shared/events"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	auction "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type Dependencies struct {
	Catalog        *catalogapp.CatalogService
	TradingUOW     tradingapp.UnitOfWork
	DealsUOW       dealsapp.UnitOfWork
	ProjectionRepo dealsapp.ProjectionRepository
	AuctionLister  ExpiredAuctionLister
	DealLister     ExpiredDealLister
	CreateAccount  *billingapp.CreateAccount
}

type Runtime struct {
	Bus           *inmemory.Bus
	Relay         *outbox.Relay
	closeAuction  *tradingapp.CloseAuction
	cancelDeal    *dealsapp.CancelDeal
	auctionLister ExpiredAuctionLister
	dealLister    ExpiredDealLister
}

type ExpiredAuctionLister interface {
	ListExpired(ctx context.Context, now time.Time, limit int) ([]tradingapp.AuctionID, error)
}

type ExpiredDealLister interface {
	ListExpiredForFallback(ctx context.Context, now time.Time, limit int) ([]string, error)
}

func New(db *sql.DB, deps Dependencies) (*Runtime, error) {
	if db == nil {
		return nil, errors.New("db is required")
	}
	if deps.Catalog == nil || deps.TradingUOW == nil || deps.DealsUOW == nil || deps.ProjectionRepo == nil || deps.AuctionLister == nil || deps.DealLister == nil {
		return nil, errors.New("runtime dependencies are incomplete")
	}

	bus := inmemory.NewBus()
	relay := outbox.NewRelay(outboxpg.NewRepository(db), DefaultDecoders())

	publishAuctionUC, err := tradingapp.NewPublishAuction(deps.TradingUOW)
	if err != nil {
		return nil, err
	}
	createProjectionUC := dealsapp.NewCreateProjection(deps.ProjectionRepo)
	createSelectionUC, err := dealsapp.NewCreateDealSelectionFromAuctionWon(deps.DealsUOW)
	if err != nil {
		return nil, err
	}
	handleDealDeclinedUC, err := dealsapp.NewHandleDealDeclined(deps.DealsUOW)
	if err != nil {
		return nil, err
	}
	closeAuctionUC, err := tradingapp.NewCloseAuction(deps.TradingUOW)
	if err != nil {
		return nil, err
	}
	cancelDealUC, err := dealsapp.NewCancelDeal(deps.DealsUOW)
	if err != nil {
		return nil, err
	}

	subscribeHandlers(bus, deps, publishAuctionUC, createProjectionUC, createSelectionUC, handleDealDeclinedUC, deps.CreateAccount) // CreateAccount optional for tests

	return &Runtime{
		Bus:           bus,
		Relay:         relay,
		closeAuction:  closeAuctionUC,
		cancelDeal:    cancelDealUC,
		auctionLister: deps.AuctionLister,
		dealLister:    deps.DealLister,
	}, nil
}

func (r *Runtime) RunCloseExpired(ctx context.Context, now time.Time, limit int) error {
	ids, err := r.auctionLister.ListExpired(ctx, now, limit)
	if err != nil {
		return err
	}
	for _, id := range ids {
		slog.InfoContext(ctx, "scheduler_close_attempt", "component", "scheduler", "operation", "close_expired_auction", "auction_id", id)
		meta := tradingapp.CommandMeta{
			CompanyID:     "system",
			UserID:        "system",
			CorrelationID: "scheduler-close",
			CausationID:   "scheduler-close",
		}
		if err := r.closeAuction.Execute(ctx, meta, id); err != nil {
			if errors.Is(err, auction.ErrAuctionNotActive) || errors.Is(err, auction.ErrInvalidStateTransition) || errors.Is(err, auction.ErrAuctionAlreadyEnded) {
				slog.InfoContext(ctx, "scheduler_close_skip", "component", "scheduler", "operation", "close_expired_auction", "auction_id", id, "reason", err.Error())
				continue
			}
			slog.ErrorContext(ctx, "scheduler_close_error", "component", "scheduler", "operation", "close_expired_auction", "auction_id", id, "error", err)
			return err
		}
		slog.InfoContext(ctx, "scheduler_close_success", "component", "scheduler", "operation", "close_expired_auction", "auction_id", id)
	}
	return nil
}

func (r *Runtime) RunCancelExpiredDeals(ctx context.Context, now time.Time, limit int) error {
	ids, err := r.dealLister.ListExpiredForFallback(ctx, now, limit)
	if err != nil {
		return err
	}
	for _, id := range ids {
		slog.InfoContext(ctx, "scheduler_cancel_deal_attempt", "component", "scheduler", "operation", "cancel_expired_deal", "deal_id", id)
		meta := dealsapp.CommandMeta{
			CompanyID:     "system",
			UserID:        "system",
			CorrelationID: "scheduler-deal-deadline",
			CausationID:   "scheduler-deal-deadline",
		}
		if err := r.cancelDeal.Execute(ctx, meta, id, "deadline exceeded"); err != nil {
			slog.ErrorContext(ctx, "scheduler_cancel_deal_error", "component", "scheduler", "operation", "cancel_expired_deal", "deal_id", id, "error", err)
			continue
		}
		slog.InfoContext(ctx, "scheduler_cancel_deal_success", "component", "scheduler", "operation", "cancel_expired_deal", "deal_id", id)
	}
	return nil
}

func subscribeHandlers(
	bus *inmemory.Bus,
	deps Dependencies,
	publishAuction *tradingapp.PublishAuction,
	createProjection *dealsapp.CreateProjection,
	createSelection *dealsapp.CreateDealSelectionFromAuctionWon,
	handleDealDeclined *dealsapp.HandleDealDeclined,
	createAccount *billingapp.CreateAccount,
) {
	bus.Subscribe("catalog.LotPublished", func(ctx context.Context, envelope events.Envelope) error {
		evt, ok := envelope.Payload.(catalog.LotPublished)
		if !ok {
			return errors.New("unexpected payload for LotPublished")
		}

		startsAt := evt.AuctionStartsAt
		endsAt := evt.AuctionEndsAt
		minBidStep := evt.MinBidStep
		if minBidStep <= 0 {
			minBidStep = 1
		}
		if startsAt.IsZero() || endsAt.IsZero() {
			return errors.New("missing auction schedule in LotPublished")
		}

		auctionID, err := auctionIDForLot(evt)
		if err != nil {
			return err
		}
		existingAuctionID := ""
		if evt.AuctionID == "" {
			existingAuctionID, err = deps.Catalog.GetLotAuctionID(ctx, evt.LotID)
			if err != nil {
				return err
			}
			if existingAuctionID != "" {
				auctionID = tradingapp.AuctionID(existingAuctionID)
			}
		}
		createAuction, err := tradingapp.NewCreateAuction(
			deps.TradingUOW,
			fixedAuctionIDFactory{auctionID: auctionID},
		)
		if err != nil {
			return err
		}

		tradingMeta := tradingMetaFromEnvelope(envelope)
		slog.InfoContext(
			ctx,
			"integration_lot_published",
			"component", "integration.runtime",
			"event_type", envelope.Type,
			"lot_id", evt.LotID,
			"auction_id", auctionID,
			"correlation_id", tradingMeta.CorrelationID,
		)
		if _, err := createAuction.Execute(ctx, tradingMeta, evt.LotID, startsAt, endsAt, evt.StartPrice, minBidStep); err != nil {
			return err
		}
		if err := publishAuction.Execute(ctx, tradingMeta, auctionID); err != nil {
			if !errors.Is(err, auction.ErrAuctionCannotBePublished) && !errors.Is(err, auction.ErrInvalidStateTransition) {
				return err
			}
		}

		dealsMeta := dealsMetaFromEnvelope(envelope)
		if evt.AuctionID == "" && existingAuctionID == "" {
			if err := deps.Catalog.AssignAuctionID(ctx, evt.LotID, string(auctionID)); err != nil {
				if !errors.Is(err, catalog.ErrAlreadyAssigned) {
					return err
				}
			}
		}
		return createProjection.Execute(
			ctx,
			dealsMeta,
			string(auctionID),
			evt.SellerCompanyID,
			dealSnapshotFromLot(evt),
			evt.StartPrice,
			envelope.OccurredAt,
		)
	})

	bus.Subscribe("trading.BidPlaced", func(ctx context.Context, envelope events.Envelope) error {
		evt, ok := envelope.Payload.(auction.BidPlaced)
		if !ok {
			return errors.New("unexpected payload for BidPlaced")
		}
		return deps.Catalog.HandleBidPlaced(ctx, catalogapp.BidPlacedDTO{
			AuctionID: evt.AuctionID,
			Amount:    evt.Amount,
		})
	})

	bus.Subscribe("trading.AuctionClosed", func(ctx context.Context, envelope events.Envelope) error {
		evt, ok := envelope.Payload.(auction.AuctionClosed)
		if !ok {
			return errors.New("unexpected payload for AuctionClosed")
		}
		return deps.Catalog.HandleAuctionClosed(ctx, catalogapp.AuctionClosedDTO{
			AuctionID: evt.AuctionID,
		})
	})

	bus.Subscribe("trading.AuctionCancelled", func(ctx context.Context, envelope events.Envelope) error {
		evt, ok := envelope.Payload.(auction.AuctionCancelled)
		if !ok {
			return errors.New("unexpected payload for AuctionCancelled")
		}
		return deps.Catalog.HandleAuctionCancelled(ctx, catalogapp.AuctionCancelledDTO{
			AuctionID: evt.AuctionID,
		})
	})

	bus.Subscribe("trading.AuctionWon", func(ctx context.Context, envelope events.Envelope) error {
		evt, ok := envelope.Payload.(auction.AuctionWon)
		if !ok {
			return errors.New("unexpected payload for AuctionWon")
		}
		if len(evt.WinnerCompanyID) == 0 {
			return errors.New("trading.AuctionWon: empty WinnerCompanyID (deal not created); check event payload / JSON field names")
		}

		dealsMeta := dealsMetaFromEnvelope(envelope)
		if err := createSelection.Execute(ctx, dealsMeta, evt.AuctionID, evt.WinnerCompanyID, evt.FinalPrice, envelope.OccurredAt); err != nil {
			return err
		}
		return deps.Catalog.HandleAuctionWon(ctx, catalogapp.AuctionWonDTO{
			AuctionID:       evt.AuctionID,
			FinalPrice:      evt.FinalPrice,
			WinnerCompanyID: evt.WinnerCompanyID[0],
		})
	})

	bus.Subscribe("deals.DealCancelled", func(ctx context.Context, envelope events.Envelope) error {
		evt, ok := envelope.Payload.(deal.DealCancelled)
		if !ok {
			return errors.New("unexpected payload for DealCancelled")
		}
		dealsMeta := dealsMetaFromEnvelope(envelope)
		err := handleDealDeclined.Execute(ctx, dealsMeta, valueOrEmpty(envelope.Meta, "auction_id"), evt.DealID)
		if err != nil {
			if errors.Is(err, dealsapp.ErrNoAvailableWinner) {
				slog.InfoContext(ctx, "integration_deal_cancelled_no_next_winner", "component", "integration.runtime", "event_type", envelope.Type, "deal_id", evt.DealID)
				return nil
			}
			return err
		}
		return nil
	})

	if createAccount != nil {
		bus.Subscribe("identity.CompanyCreated", func(ctx context.Context, envelope events.Envelope) error {
			evt, ok := envelope.Payload.(identity.CompanyCreated)
			if !ok {
				return errors.New("unexpected payload for CompanyCreated")
			}
			slog.InfoContext(ctx, "integration_company_created", "component", "integration.runtime", "company_id", evt.CompanyID)
			return createAccount.Execute(ctx, evt.CompanyID)
		})
	}
}

func DefaultDecoders() map[string]outbox.Decoder {
	return map[string]outbox.Decoder{
		"catalog.ProductCreated":          outbox.JSONDecoder[catalog.ProductCreated](),
		"catalog.ProductUpdated":          outbox.JSONDecoder[catalog.ProductUpdated](),
		"catalog.ProductPublished":        outbox.JSONDecoder[catalog.ProductPublished](),
		"catalog.ProductUnpublished":      outbox.JSONDecoder[catalog.ProductUnpublished](),
		"catalog.LotCreated":              outbox.JSONDecoder[catalog.LotCreated](),
		"catalog.LotPublished":            outbox.JSONDecoder[catalog.LotPublished](),
		"catalog.LotUnpublished":          outbox.JSONDecoder[catalog.LotUnpublished](),
		"catalog.LotClosed":               outbox.JSONDecoder[catalog.LotClosed](),
		"catalog.LotPriceUpdated":         outbox.JSONDecoder[catalog.LotPriceUpdated](),
		"catalog.LotAuctionLinked":        outbox.JSONDecoder[catalog.LotAuctionLinked](),
		"trading.AuctionPublished":        outbox.JSONDecoder[auction.AuctionPublished](),
		"trading.BidPlaced":               outbox.JSONDecoder[auction.BidPlaced](),
		"trading.AuctionClosed":           outbox.JSONDecoder[auction.AuctionClosed](),
		"trading.AuctionWon":              outbox.JSONDecoder[auction.AuctionWon](),
		"trading.AuctionCancelled":        outbox.JSONDecoder[auction.AuctionCancelled](),
		"deals.DealCreated":               outbox.JSONDecoder[deal.DealCreated](),
		"deals.DealConfirmationRequested": outbox.JSONDecoder[deal.DealConfirmationRequested](),
		"deals.DealConfirmationApproved":  outbox.JSONDecoder[deal.DealConfirmationApproved](),
		"deals.DealConfirmationRejected":  outbox.JSONDecoder[deal.DealConfirmationRejected](),
		"deals.DealConfirmed":             outbox.JSONDecoder[deal.DealConfirmed](),
		"deals.ContractPrepared":          outbox.JSONDecoder[deal.ContractPrepared](),
		"deals.ContractSigned":            outbox.JSONDecoder[deal.ContractSigned](),
		"deals.PaymentRequested":          outbox.JSONDecoder[deal.PaymentRequested](),
		"deals.DealPaid":                  outbox.JSONDecoder[deal.DealPaid](),
		"deals.ShipmentRequested":         outbox.JSONDecoder[deal.ShipmentRequested](),
		"deals.DealShipped":               outbox.JSONDecoder[deal.DealShipped](),
		"deals.DealCompleted":             outbox.JSONDecoder[deal.DealCompleted](),
		"deals.DealCancelled":             outbox.JSONDecoder[deal.DealCancelled](),
		"deals.PriceUpdated":              outbox.JSONDecoder[deal.PriceUpdated](),
		"identity.CompanyCreated":         outbox.JSONDecoder[identity.CompanyCreated](),
	}
}

type fixedAuctionIDFactory struct {
	auctionID tradingapp.AuctionID
}

func (f fixedAuctionIDFactory) NewID() (tradingapp.AuctionID, error) {
	return f.auctionID, nil
}

func tradingMetaFromEnvelope(envelope events.Envelope) tradingapp.CommandMeta {
	return tradingapp.CommandMeta{
		CompanyID:     valueOrEmpty(envelope.Meta, "company_id"),
		UserID:        valueOrEmpty(envelope.Meta, "user_id"),
		CorrelationID: valueOrEmpty(envelope.Meta, "correlation_id"),
		CausationID:   valueOrEmpty(envelope.Meta, "causation_id"),
	}
}

func dealsMetaFromEnvelope(envelope events.Envelope) dealsapp.CommandMeta {
	return dealsapp.CommandMeta{
		CompanyID:     valueOrEmpty(envelope.Meta, "company_id"),
		UserID:        valueOrEmpty(envelope.Meta, "user_id"),
		CorrelationID: valueOrEmpty(envelope.Meta, "correlation_id"),
		CausationID:   valueOrEmpty(envelope.Meta, "causation_id"),
	}
}

func valueOrEmpty(meta map[string]string, key string) string {
	if meta == nil {
		return ""
	}
	return meta[key]
}

func dealSnapshotFromLot(evt catalog.LotPublished) deal.ProductSnapshot {
	return deal.ProductSnapshot{
		ProductID:      evt.Product.ProductID,
		Name:           evt.Product.Name,
		Weight:         evt.Product.Weight,
		Unit:           evt.Product.Unit,
		Size:           evt.Product.Size,
		ProcessingType: string(evt.Product.ProcessingType),
	}
}

func auctionIDForLot(evt catalog.LotPublished) (tradingapp.AuctionID, error) {
	if evt.AuctionID != "" {
		return tradingapp.AuctionID(evt.AuctionID), nil
	}
	return tradingapp.RandomAuctionIDFactory{}.NewID()
}
