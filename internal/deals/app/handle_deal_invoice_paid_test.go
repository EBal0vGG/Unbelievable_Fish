package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

func TestHandleDealInvoicePaid_happyPath_emitsPaidAndFinalized(t *testing.T) {
	d, sel := paymentRequestedDealAndSelection(t)
	calls := []string{}
	dealsSpy := &dealRepoSpy{calls: &calls, deal: d}
	selections := &selectionRepoSpy{calls: &calls, selection: sel}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{
		deals:         dealsSpy,
		confirmations: &confirmationRepoSpy{},
		projections:   &projectionRepoSpy{},
		selections:    selections,
		outbox:        outbox,
	}}

	uc, err := NewHandleDealInvoicePaid(uow)
	if err != nil {
		t.Fatal(err)
	}
	goods := d.CalculateTotal()
	fee := goods * 3 / 100
	evt := wallet.DealInvoicePaid{
		InvoiceID:            "inv-1",
		DealID:               d.ID(),
		AuctionID:            d.AuctionID(),
		BuyerCompanyID:       d.CustomerID(),
		GoodsAmount:          goods,
		PlatformFeeDueAmount: fee,
		Amount:               goods + fee,
		Currency:             wallet.CurrencyRUB,
		PaidAt:               time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
	}
	if err := uc.Execute(context.Background(), evt); err != nil {
		t.Fatal(err)
	}
	if d.Status() != deal.DealStatusPaid {
		t.Fatalf("deal status: want paid got %s", d.Status())
	}
	if selections.lastSaved == nil || selections.lastSaved.Status != deal.WinnerSelectionStatusFinalized {
		t.Fatalf("selection not finalized: %+v", selections.lastSaved)
	}
	assertCalls(t, calls, []string{"load_deal_for_update", "load_selection_for_update", "save_deal", "save_selection", "outbox"})
	if len(outbox.saved) != 1 {
		t.Fatalf("expected one outbox batch, got %d", len(outbox.saved))
	}
	batch := outbox.saved[0]
	var seenPaid, seenFinal bool
	for _, e := range batch {
		switch e.(type) {
		case deal.DealPaid:
			seenPaid = true
		case deal.WinnerSelectionFinalized:
			seenFinal = true
		}
	}
	if !seenPaid || !seenFinal {
		t.Fatalf("outbox batch missing events: %#v", batch)
	}
}

func TestHandleDealInvoicePaid_alreadyPaid_noOutbox(t *testing.T) {
	d, sel := paymentRequestedDealAndSelection(t)
	if _, err := d.MarkAsPaid("x", "invoice"); err != nil {
		t.Fatal(err)
	}
	calls := []string{}
	uow := &spyUOW{tx: &spyTx{
		deals:         &dealRepoSpy{calls: &calls, deal: d},
		confirmations: &confirmationRepoSpy{},
		projections:   &projectionRepoSpy{},
		selections:    &selectionRepoSpy{selection: sel},
		outbox:        &outboxSpy{calls: &calls},
	}}
	uc, err := NewHandleDealInvoicePaid(uow)
	if err != nil {
		t.Fatal(err)
	}
	goods := d.CalculateTotal()
	if err := uc.Execute(context.Background(), wallet.DealInvoicePaid{
		InvoiceID:   "inv-1",
		DealID:      d.ID(),
		AuctionID:   d.AuctionID(),
		BuyerCompanyID: d.CustomerID(),
		GoodsAmount: goods,
		Amount:      goods,
		PaidAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	assertCalls(t, calls, []string{"load_deal_for_update"})
}

func TestHandleDealInvoicePaid_cancelled(t *testing.T) {
	d, sel := paymentRequestedDealAndSelection(t)
	if _, err := d.Cancel("x", d.CustomerID()); err != nil {
		t.Fatal(err)
	}
	uow := &spyUOW{tx: &spyTx{
		deals:         &dealRepoSpy{deal: d},
		confirmations: &confirmationRepoSpy{},
		projections:   &projectionRepoSpy{},
		selections:    &selectionRepoSpy{selection: sel},
		outbox:        &outboxSpy{},
	}}
	uc, err := NewHandleDealInvoicePaid(uow)
	if err != nil {
		t.Fatal(err)
	}
	err = uc.Execute(context.Background(), wallet.DealInvoicePaid{DealID: d.ID(), AuctionID: d.AuctionID(), BuyerCompanyID: d.CustomerID(), PaidAt: time.Now().UTC()})
	if !errors.Is(err, deal.ErrDealCancelledPayment) {
		t.Fatalf("want ErrDealCancelledPayment got %v", err)
	}
}

func TestHandleDealInvoicePaid_wrongBuyer(t *testing.T) {
	d, sel := paymentRequestedDealAndSelection(t)
	uow := &spyUOW{tx: &spyTx{
		deals:         &dealRepoSpy{deal: d},
		confirmations: &confirmationRepoSpy{},
		projections:   &projectionRepoSpy{},
		selections:    &selectionRepoSpy{selection: sel},
		outbox:        &outboxSpy{},
	}}
	uc, err := NewHandleDealInvoicePaid(uow)
	if err != nil {
		t.Fatal(err)
	}
	err = uc.Execute(context.Background(), wallet.DealInvoicePaid{
		DealID:         d.ID(),
		AuctionID:      d.AuctionID(),
		BuyerCompanyID: "someone-else",
		PaidAt:         time.Now().UTC(),
	})
	if !errors.Is(err, deal.ErrNotDealParticipant) {
		t.Fatalf("want ErrNotDealParticipant got %v", err)
	}
}

func TestHandleDealInvoicePaid_invoiceInvariant(t *testing.T) {
	d, sel := paymentRequestedDealAndSelection(t)
	uow := &spyUOW{tx: &spyTx{
		deals:         &dealRepoSpy{deal: d},
		confirmations: &confirmationRepoSpy{},
		projections:   &projectionRepoSpy{},
		selections:    &selectionRepoSpy{selection: sel},
		outbox:        &outboxSpy{},
	}}
	uc, err := NewHandleDealInvoicePaid(uow)
	if err != nil {
		t.Fatal(err)
	}
	goods := d.CalculateTotal()
	err = uc.Execute(context.Background(), wallet.DealInvoicePaid{
		DealID:               d.ID(),
		AuctionID:            d.AuctionID(),
		BuyerCompanyID:       d.CustomerID(),
		GoodsAmount:          goods,
		PlatformFeeDueAmount: 1,
		Amount:               goods + 99,
		PaidAt:               time.Now().UTC(),
	})
	if !errors.Is(err, deal.ErrDealInvoicePaidInvariant) {
		t.Fatalf("want ErrDealInvoicePaidInvariant got %v", err)
	}
}

func TestHandleDealInvoicePaid_selectionNotAwaitingPayment(t *testing.T) {
	d, sel := paymentRequestedDealAndSelection(t)
	sel.Status = deal.WinnerSelectionActive
	uow := &spyUOW{tx: &spyTx{
		deals:         &dealRepoSpy{deal: d},
		confirmations: &confirmationRepoSpy{},
		projections:   &projectionRepoSpy{},
		selections:    &selectionRepoSpy{selection: sel},
		outbox:        &outboxSpy{},
	}}
	uc, err := NewHandleDealInvoicePaid(uow)
	if err != nil {
		t.Fatal(err)
	}
	goods := d.CalculateTotal()
	err = uc.Execute(context.Background(), wallet.DealInvoicePaid{
		InvoiceID:     "inv-1",
		DealID:        d.ID(),
		AuctionID:     d.AuctionID(),
		BuyerCompanyID: d.CustomerID(),
		GoodsAmount:   goods,
		Amount:        goods,
		PaidAt:        time.Now().UTC(),
	})
	if !errors.Is(err, deal.ErrWinnerSelectionNotAwaitingPayment) {
		t.Fatalf("want ErrWinnerSelectionNotAwaitingPayment got %v", err)
	}
}

func TestHandleDealInvoicePaid_selectionAlreadyFinalized_emitsPaidOnly(t *testing.T) {
	d, sel := paymentRequestedDealAndSelection(t)
	if err := sel.MarkFinalized(d.ID()); err != nil {
		t.Fatal(err)
	}
	calls := []string{}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{
		deals:         &dealRepoSpy{calls: &calls, deal: d},
		confirmations: &confirmationRepoSpy{},
		projections:   &projectionRepoSpy{},
		selections:    &selectionRepoSpy{selection: sel},
		outbox:        outbox,
	}}
	uc, err := NewHandleDealInvoicePaid(uow)
	if err != nil {
		t.Fatal(err)
	}
	goods := d.CalculateTotal()
	if err := uc.Execute(context.Background(), wallet.DealInvoicePaid{
		InvoiceID:     "inv-1",
		DealID:        d.ID(),
		AuctionID:     d.AuctionID(),
		BuyerCompanyID: d.CustomerID(),
		GoodsAmount:   goods,
		Amount:        goods,
		PaidAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if len(outbox.saved) != 1 {
		t.Fatalf("batches: %d", len(outbox.saved))
	}
	for _, e := range outbox.saved[0] {
		if _, ok := e.(deal.WinnerSelectionFinalized); ok {
			t.Fatal("unexpected WinnerSelectionFinalized when selection already finalized")
		}
	}
}

func paymentRequestedDealAndSelection(t *testing.T) (*deal.Deal, *deal.WinnerSelection) {
	t.Helper()
	item := createPendingDeal(t)
	if _, err := item.Confirm(); err != nil {
		t.Fatal(err)
	}
	if _, err := item.PrepareContract("C-1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := item.SignContract("buyer", "sig"); err != nil {
		t.Fatal(err)
	}
	if _, err := item.RequestPayment("", nil); err != nil {
		t.Fatal(err)
	}
	sel := deal.NewWinnerSelection(
		item.AuctionID(),
		[]string{item.CustomerID()},
		item.UnitPrice(),
		time.Now().UTC(),
		item.SupplierID(),
		item.ProductSnapshot(),
	)
	sel.DealID = item.ID()
	sel.Status = deal.WinnerSelectionConfirmedPendingPayment
	return item, sel
}
