package runtime

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalog "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	deal "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/eventbus/inmemory"
	outbox "github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox"
	outboxpg "github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox/postgres"
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
	BillingTx      billingapp.UnitOfWork
	CreateAccount  *billingapp.CreateAccount
	// ReleaseAuctionDepositsExceptCandidates releases HELD deposits for bidders not in AuctionWon.WinnerCompanyID (nil = skip).
	ReleaseAuctionDepositsExceptCandidates *billingapp.ReleaseAuctionDepositsExceptCandidates
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

	registerIntegrationHandlers(bus, deps, publishAuctionUC, createProjectionUC, createSelectionUC, handleDealDeclinedUC, deps.CreateAccount)

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
		"billing.AuctionDepositReleased":  outbox.JSONDecoder[wallet.AuctionDepositReleased](),
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
