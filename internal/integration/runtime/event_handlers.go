package runtime

import (
	"context"
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
	"github.com/EBal0vGG/Unbelievable_Fish/internal/shared/events"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	auction "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

// lotPublishedHandler is integration policy for catalog.LotPublished (trading auction + publish + catalog assign + deal projection).
type lotPublishedHandler struct {
	deps              Dependencies
	publishAuction    *tradingapp.PublishAuction
	createProjection  *dealsapp.CreateProjection
}

func newLotPublishedHandler(
	deps Dependencies,
	publishAuction *tradingapp.PublishAuction,
	createProjection *dealsapp.CreateProjection,
) *lotPublishedHandler {
	return &lotPublishedHandler{
		deps:             deps,
		publishAuction:   publishAuction,
		createProjection: createProjection,
	}
}

func (h *lotPublishedHandler) Execute(ctx context.Context, envelope events.Envelope) error {
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
		existingAuctionID, err = h.deps.Catalog.GetLotAuctionID(ctx, evt.LotID)
		if err != nil {
			return err
		}
		if existingAuctionID != "" {
			auctionID = tradingapp.AuctionID(existingAuctionID)
		}
	}
	createAuction, err := tradingapp.NewCreateAuction(
		h.deps.TradingUOW,
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
	if err := h.publishAuction.Execute(ctx, tradingMeta, auctionID); err != nil {
		if !errors.Is(err, auction.ErrAuctionCannotBePublished) && !errors.Is(err, auction.ErrInvalidStateTransition) {
			return err
		}
	}

	dealsMeta := dealsMetaFromEnvelope(envelope)
	if evt.AuctionID == "" && existingAuctionID == "" {
		if err := h.deps.Catalog.AssignAuctionID(ctx, evt.LotID, string(auctionID)); err != nil {
			if !errors.Is(err, catalog.ErrAlreadyAssigned) {
				return err
			}
		}
	}
	return h.createProjection.Execute(
		ctx,
		dealsMeta,
		string(auctionID),
		evt.SellerCompanyID,
		dealSnapshotFromLot(evt),
		evt.StartPrice,
		envelope.OccurredAt,
	)
}

// handleCatalogFromTradingEvents syncs catalog read-model from trading outbox events (thin delegates to CatalogService).
type handleCatalogFromTradingEvents struct {
	catalog *catalogapp.CatalogService
}

func newHandleCatalogFromTradingEvents(catalog *catalogapp.CatalogService) *handleCatalogFromTradingEvents {
	return &handleCatalogFromTradingEvents{catalog: catalog}
}

func (h *handleCatalogFromTradingEvents) ExecuteBidPlaced(ctx context.Context, envelope events.Envelope) error {
	evt, ok := envelope.Payload.(auction.BidPlaced)
	if !ok {
		return errors.New("unexpected payload for BidPlaced")
	}
	return h.catalog.HandleBidPlaced(ctx, catalogapp.BidPlacedDTO{
		AuctionID: evt.AuctionID,
		Amount:    evt.Amount,
	})
}

func (h *handleCatalogFromTradingEvents) ExecuteAuctionClosed(ctx context.Context, envelope events.Envelope) error {
	evt, ok := envelope.Payload.(auction.AuctionClosed)
	if !ok {
		return errors.New("unexpected payload for AuctionClosed")
	}
	return h.catalog.HandleAuctionClosed(ctx, catalogapp.AuctionClosedDTO{
		AuctionID: evt.AuctionID,
	})
}

func (h *handleCatalogFromTradingEvents) ExecuteAuctionCancelled(ctx context.Context, envelope events.Envelope) error {
	evt, ok := envelope.Payload.(auction.AuctionCancelled)
	if !ok {
		return errors.New("unexpected payload for AuctionCancelled")
	}
	return h.catalog.HandleAuctionCancelled(ctx, catalogapp.AuctionCancelledDTO{
		AuctionID: evt.AuctionID,
	})
}

// auctionWonHandler runs deal selection, catalog sync, then billing deposit release (integration policy).
type auctionWonHandler struct {
	deps            Dependencies
	createSelection *dealsapp.CreateDealSelectionFromAuctionWon
}

func newAuctionWonHandler(deps Dependencies, createSelection *dealsapp.CreateDealSelectionFromAuctionWon) *auctionWonHandler {
	return &auctionWonHandler{deps: deps, createSelection: createSelection}
}

func (h *auctionWonHandler) Execute(ctx context.Context, envelope events.Envelope) error {
	evt, ok := envelope.Payload.(auction.AuctionWon)
	if !ok {
		return errors.New("unexpected payload for AuctionWon")
	}
	if len(evt.WinnerCompanyID) == 0 {
		return errors.New("trading.AuctionWon: empty WinnerCompanyID (deal not created); check event payload / JSON field names")
	}

	dealsMeta := dealsMetaFromEnvelope(envelope)
	if err := h.createSelection.Execute(ctx, dealsMeta, evt.AuctionID, evt.WinnerCompanyID, evt.FinalPrice, envelope.OccurredAt); err != nil {
		return err
	}
	if err := h.deps.Catalog.HandleAuctionWon(ctx, catalogapp.AuctionWonDTO{
		AuctionID:       evt.AuctionID,
		FinalPrice:      evt.FinalPrice,
		WinnerCompanyID: evt.WinnerCompanyID[0],
	}); err != nil {
		return err
	}
	if h.deps.ReleaseAuctionDepositsExceptCandidates != nil && h.deps.BillingTx != nil {
		return h.deps.BillingTx.WithinTx(ctx, func(txCtx context.Context) error {
			return h.deps.ReleaseAuctionDepositsExceptCandidates.Execute(txCtx, evt.AuctionID, evt.WinnerCompanyID, "LOST_AUCTION")
		})
	}
	return nil
}

type dealCancelledHandler struct {
	handleDealDeclined *dealsapp.HandleDealDeclined
}

func newDealCancelledHandler(uc *dealsapp.HandleDealDeclined) *dealCancelledHandler {
	return &dealCancelledHandler{handleDealDeclined: uc}
}

type winnerRejectedHandler struct {
	deps Dependencies
}

func newWinnerRejectedHandler(deps Dependencies) *winnerRejectedHandler {
	return &winnerRejectedHandler{deps: deps}
}

func (h *winnerRejectedHandler) Execute(ctx context.Context, envelope events.Envelope) error {
	evt, ok := envelope.Payload.(deal.WinnerRejected)
	if !ok {
		return errors.New("unexpected payload for WinnerRejected")
	}
	if h.deps.CaptureAuctionDeposit == nil || h.deps.BillingTx == nil {
		return errors.New("billing dependencies are required for WinnerRejected handling")
	}
	slog.InfoContext(ctx, "integration_winner_rejected", "component", "integration.runtime", "auction_id", evt.AuctionID, "company_id", evt.CompanyID, "deal_id", evt.DealID)
	return h.deps.BillingTx.WithinTx(ctx, func(txCtx context.Context) error {
		return h.deps.CaptureAuctionDeposit.Execute(txCtx, evt.CompanyID, evt.AuctionID, evt.Reason)
	})
}

func (h *dealCancelledHandler) Execute(ctx context.Context, envelope events.Envelope) error {
	evt, ok := envelope.Payload.(deal.DealCancelled)
	if !ok {
		return errors.New("unexpected payload for DealCancelled")
	}
	dealsMeta := dealsMetaFromEnvelope(envelope)
	err := h.handleDealDeclined.Execute(ctx, dealsMeta, valueOrEmpty(envelope.Meta, "auction_id"), evt.DealID)
	if err != nil {
		if errors.Is(err, dealsapp.ErrNoAvailableWinner) {
			slog.InfoContext(ctx, "integration_deal_cancelled_no_next_winner", "component", "integration.runtime", "event_type", envelope.Type, "deal_id", evt.DealID)
			return nil
		}
		return err
	}
	return nil
}

type companyCreatedHandler struct {
	deps          Dependencies
	createAccount *billingapp.CreateAccount
}

func newCompanyCreatedHandler(deps Dependencies, createAccount *billingapp.CreateAccount) *companyCreatedHandler {
	return &companyCreatedHandler{deps: deps, createAccount: createAccount}
}

func (h *companyCreatedHandler) Execute(ctx context.Context, envelope events.Envelope) error {
	evt, ok := envelope.Payload.(identity.CompanyCreated)
	if !ok {
		return errors.New("unexpected payload for CompanyCreated")
	}
	slog.InfoContext(ctx, "integration_company_created", "component", "integration.runtime", "company_id", evt.CompanyID)
	if h.deps.BillingTx == nil {
		return errors.New("billing transaction manager is required for CompanyCreated handling")
	}
	return h.deps.BillingTx.WithinTx(ctx, func(txCtx context.Context) error {
		return h.createAccount.Execute(txCtx, evt.CompanyID)
	})
}

func registerIntegrationHandlers(
	bus *inmemory.Bus,
	deps Dependencies,
	publishAuction *tradingapp.PublishAuction,
	createProjection *dealsapp.CreateProjection,
	createSelection *dealsapp.CreateDealSelectionFromAuctionWon,
	handleDealDeclined *dealsapp.HandleDealDeclined,
	createAccount *billingapp.CreateAccount,
) {
	if deps.Catalog != nil && publishAuction != nil && createProjection != nil {
		lotH := newLotPublishedHandler(deps, publishAuction, createProjection)
		bus.Subscribe("catalog.LotPublished", lotH.Execute)
	}
	if deps.Catalog != nil {
		catH := newHandleCatalogFromTradingEvents(deps.Catalog)
		bus.Subscribe("trading.BidPlaced", catH.ExecuteBidPlaced)
		bus.Subscribe("trading.AuctionClosed", catH.ExecuteAuctionClosed)
		bus.Subscribe("trading.AuctionCancelled", catH.ExecuteAuctionCancelled)
	}
	if deps.Catalog != nil && createSelection != nil {
		wonH := newAuctionWonHandler(deps, createSelection)
		bus.Subscribe("trading.AuctionWon", wonH.Execute)
	}
	if handleDealDeclined != nil {
		dealH := newDealCancelledHandler(handleDealDeclined)
		bus.Subscribe("deals.DealCancelled", dealH.Execute)
	}
	if deps.CaptureAuctionDeposit != nil && deps.BillingTx != nil {
		wrH := newWinnerRejectedHandler(deps)
		bus.Subscribe("deals.WinnerRejected", wrH.Execute)
	}
	if createAccount != nil {
		ccH := newCompanyCreatedHandler(deps, createAccount)
		bus.Subscribe("identity.CompanyCreated", ccH.Execute)
	}
	if deps.CreateDealInvoice != nil && deps.BillingTx != nil {
		prH := newPaymentRequestedHandler(deps)
		bus.Subscribe("deals.PaymentRequested", prH.Execute)
	}
	if deps.HandleDealInvoicePaid != nil {
		dipH := newDealInvoicePaidHandler(deps)
		bus.Subscribe("billing.DealInvoicePaid", dipH.Execute)
	}
	if deps.SettleWinnerDepositAfterInvoicePaid != nil && deps.ReleaseAuctionDepositsExceptCandidates != nil && deps.BillingTx != nil {
		wsfH := newWinnerSelectionFinalizedHandler(deps)
		bus.Subscribe("deals.WinnerSelectionFinalized", wsfH.Execute)
	}
}

type paymentRequestedHandler struct {
	deps Dependencies
}

func newPaymentRequestedHandler(deps Dependencies) *paymentRequestedHandler {
	return &paymentRequestedHandler{deps: deps}
}

func (h *paymentRequestedHandler) Execute(ctx context.Context, envelope events.Envelope) error {
	evt, ok := envelope.Payload.(deal.PaymentRequested)
	if !ok {
		return errors.New("unexpected payload for PaymentRequested")
	}
	cur := wallet.Currency(evt.Currency)
	if evt.Currency == "" {
		cur = wallet.CurrencyRUB
	}
	var due time.Time
	if evt.DueDate != nil {
		due = *evt.DueDate
	}
	return h.deps.BillingTx.WithinTx(ctx, func(txCtx context.Context) error {
		_, err := h.deps.CreateDealInvoice.Execute(txCtx, billingapp.CreateDealInvoiceCommand{
			DealID:          evt.DealID,
			AuctionID:       evt.AuctionID,
			BuyerCompanyID:  evt.BuyerCompanyID,
			SellerCompanyID: evt.SellerCompanyID,
			GoodsAmount:     evt.GoodsAmount,
			Currency:        cur,
			DueAt:           due,
		})
		return err
	})
}

type dealInvoicePaidHandler struct {
	deps Dependencies
}

func newDealInvoicePaidHandler(deps Dependencies) *dealInvoicePaidHandler {
	return &dealInvoicePaidHandler{deps: deps}
}

func (h *dealInvoicePaidHandler) Execute(ctx context.Context, envelope events.Envelope) error {
	evt, ok := envelope.Payload.(wallet.DealInvoicePaid)
	if !ok {
		return errors.New("unexpected payload for DealInvoicePaid")
	}
	if h.deps.HandleDealInvoicePaid == nil {
		return errors.New("HandleDealInvoicePaid is not configured")
	}
	return h.deps.HandleDealInvoicePaid.Execute(ctx, evt)
}

type winnerSelectionFinalizedHandler struct {
	deps Dependencies
}

func newWinnerSelectionFinalizedHandler(deps Dependencies) *winnerSelectionFinalizedHandler {
	return &winnerSelectionFinalizedHandler{deps: deps}
}

func (h *winnerSelectionFinalizedHandler) Execute(ctx context.Context, envelope events.Envelope) error {
	evt, ok := envelope.Payload.(deal.WinnerSelectionFinalized)
	if !ok {
		return errors.New("unexpected payload for WinnerSelectionFinalized")
	}
	if h.deps.BillingTx == nil || h.deps.SettleWinnerDepositAfterInvoicePaid == nil || h.deps.ReleaseAuctionDepositsExceptCandidates == nil {
		return errors.New("billing settlement dependencies are not configured")
	}
	return h.deps.BillingTx.WithinTx(ctx, func(txCtx context.Context) error {
		if err := h.deps.SettleWinnerDepositAfterInvoicePaid.Execute(txCtx, evt.AuctionID, evt.CompanyID, evt.GoodsAmount, evt.PlatformFeeDueAmount, "WINNER_FINALIZED"); err != nil {
			return err
		}
		return h.deps.ReleaseAuctionDepositsExceptCandidates.Execute(txCtx, evt.AuctionID, []string{evt.CompanyID}, "WINNER_FINALIZED")
	})
}
