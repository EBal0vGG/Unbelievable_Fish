package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type CreateDealInvoice struct {
	invoices             DealInvoiceRepository
	deposits             AuctionDepositRepository
	provider             PaymentProvider
	providerName         string
	ids                  IDGenerator
	clock                Clock
	events               DomainEventPublisher
	publicBillingBaseURL string
}

type CreateDealInvoiceCommand struct {
	DealID          string
	AuctionID       string
	BuyerCompanyID  string
	SellerCompanyID string
	GoodsAmount     int64
	Currency        wallet.Currency
	DueAt           time.Time
}

func NewCreateDealInvoice(
	invoices DealInvoiceRepository,
	deposits AuctionDepositRepository,
	provider PaymentProvider,
	providerName string,
	ids IDGenerator,
	clock Clock,
	events DomainEventPublisher,
	publicBillingBaseURL string,
) (*CreateDealInvoice, error) {
	if invoices == nil || deposits == nil || provider == nil {
		return nil, ErrNilDependency
	}
	if isBlank(providerName) {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &CreateDealInvoice{
		invoices:             invoices,
		deposits:             deposits,
		provider:             provider,
		providerName:         providerName,
		ids:                  ids,
		clock:                clock,
		events:               events,
		publicBillingBaseURL: publicBillingBaseURL,
	}, nil
}

func (uc *CreateDealInvoice) Execute(ctx context.Context, cmd CreateDealInvoiceCommand) (*wallet.DealInvoice, error) {
	if isBlank(cmd.DealID) || isBlank(cmd.BuyerCompanyID) || isBlank(cmd.SellerCompanyID) {
		return nil, wallet.ErrInvalidIdentifier
	}
	if cmd.BuyerCompanyID == cmd.SellerCompanyID {
		return nil, wallet.ErrInvalidDealInvoice
	}
	if cmd.GoodsAmount <= 0 {
		return nil, wallet.ErrInvalidAmount
	}
	if cmd.Currency != wallet.CurrencyRUB {
		return nil, wallet.ErrUnsupportedCurrency
	}

	existing, err := uc.invoices.LoadByDealID(ctx, cmd.DealID)
	if err == nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, ErrDealInvoiceNotFound) {
		return nil, err
	}

	held := int64(0)
	if !isBlank(cmd.AuctionID) {
		dep, derr := uc.deposits.Find(ctx, cmd.AuctionID, cmd.BuyerCompanyID)
		if derr == nil && dep.Status == wallet.DepositHeld {
			held = dep.Amount
		}
	}
	fee := platformFeeFromFinalPrice(cmd.GoodsAmount)
	feeDue := fee - held
	if feeDue < 0 {
		feeDue = 0
	}

	now := uc.clock.Now()
	due := cmd.DueAt
	if due.IsZero() {
		due = now.Add(24 * time.Hour)
	}
	if due.Before(now) {
		slog.DebugContext(ctx, "deal_invoice_due_clamped_to_default",
			"component", "billing.create_deal_invoice",
			"deal_id", cmd.DealID,
			"requested_due_before_now", true,
			"effective_due", now.Add(24*time.Hour),
		)
		due = now.Add(24 * time.Hour)
	}

	invID := uc.ids.NewID()
	inv, err := wallet.NewDealInvoice(
		invID, cmd.DealID, cmd.AuctionID, cmd.BuyerCompanyID, cmd.SellerCompanyID,
		cmd.GoodsAmount, feeDue, cmd.Currency, uc.providerName, due, now,
	)
	if err != nil {
		return nil, err
	}
	if err := uc.invoices.Create(ctx, inv); err != nil {
		return nil, err
	}

	resp, err := uc.provider.CreateDealInvoice(ctx, CreateDealInvoiceRequest{
		InvoiceID:       invID,
		DealID:          cmd.DealID,
		BuyerCompanyID:  cmd.BuyerCompanyID,
		SellerCompanyID: cmd.SellerCompanyID,
		Amount:          inv.TotalAmount,
		Currency:        string(cmd.Currency),
		DueAt:           due,
		ReturnURL:       uc.publicBillingBaseURL,
	})
	if err != nil {
		return nil, err
	}
	if err := inv.AttachProvider(resp.ProviderInvoiceID, resp.PaymentURL); err != nil {
		return nil, err
	}
	if err := uc.invoices.Save(ctx, inv); err != nil {
		return nil, err
	}

	if uc.events != nil {
		_ = uc.events.Publish(ctx, inv.DealID, inv.BuyerCompanyID, wallet.DealInvoiceCreated{
			InvoiceID:            inv.ID,
			DealID:               inv.DealID,
			AuctionID:            inv.AuctionID,
			BuyerCompanyID:       inv.BuyerCompanyID,
			SellerCompanyID:      inv.SellerCompanyID,
			GoodsAmount:          inv.GoodsAmount,
			PlatformFeeDueAmount: inv.PlatformFeeDueAmount,
			TotalAmount:          inv.TotalAmount,
			Currency:             inv.Currency,
			PaymentURL:           inv.PaymentURL,
			DueAt:                inv.DueAt,
			CreatedAt:            inv.CreatedAt,
		})
	}
	return inv, nil
}
