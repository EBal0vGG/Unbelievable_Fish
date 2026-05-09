package app

import (
	"context"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type mapDealRepo struct {
	byID map[string]*deal.Deal
}

func (m *mapDealRepo) Save(_ context.Context, item *deal.Deal) error {
	m.byID[item.ID()] = item
	return nil
}

func (m *mapDealRepo) GetByID(_ context.Context, dealID string) (*deal.Deal, error) {
	d, ok := m.byID[dealID]
	if !ok {
		return nil, ErrDealNotFound
	}
	return d, nil
}

func (m *mapDealRepo) GetByIDForUpdate(ctx context.Context, dealID string) (*deal.Deal, error) {
	return m.GetByID(ctx, dealID)
}

func (m *mapDealRepo) GetActiveDealByAuctionID(_ context.Context, auctionID string) (*deal.Deal, error) {
	for _, d := range m.byID {
		if d.AuctionID() == auctionID && d.Status() != deal.DealStatusCancelled {
			return d, nil
		}
	}
	return nil, ErrDealNotFound
}

type mapSelectionRepo struct {
	byAuction map[string]*deal.WinnerSelection
}

func (m *mapSelectionRepo) Save(_ context.Context, item *deal.WinnerSelection) error {
	m.byAuction[item.AuctionID] = item
	return nil
}

func (m *mapSelectionRepo) GetByAuctionID(_ context.Context, auctionID string) (*deal.WinnerSelection, error) {
	s, ok := m.byAuction[auctionID]
	if !ok {
		return nil, deal.ErrSelectionNotFound
	}
	return s, nil
}

func (m *mapSelectionRepo) GetByAuctionIDForUpdate(ctx context.Context, auctionID string) (*deal.WinnerSelection, error) {
	return m.GetByAuctionID(ctx, auctionID)
}

type sliceOutbox struct {
	saved [][]deal.Event
}

func (o *sliceOutbox) Add(_ context.Context, events []deal.Event) error {
	o.saved = append(o.saved, events)
	return nil
}

type expFixedClock struct{ t time.Time }

func (c expFixedClock) Now() time.Time { return c.t }

func paymentTimeoutDealTwoWinners(t *testing.T) (*deal.Deal, *deal.WinnerSelection) {
	t.Helper()
	fac := deal.NewFactory()
	snap := deal.ProductSnapshot{ProductID: "p1", Name: "Fish"}
	wonAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	d1, _, err := fac.CreateFromSelection("auc-pt", "sup-1", snap, "buyer-1", 200, wonAt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d1.Confirm(); err != nil {
		t.Fatal(err)
	}
	if _, err := d1.PrepareContract("C-1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := d1.SignContract("buyer", "sig"); err != nil {
		t.Fatal(err)
	}
	if _, err := d1.RequestPayment("", nil); err != nil {
		t.Fatal(err)
	}
	sel := deal.NewWinnerSelection("auc-pt", []string{"buyer-1", "buyer-2"}, 200, wonAt, "sup-1", snap)
	sel.DealID = d1.ID()
	sel.Status = deal.WinnerSelectionConfirmedPendingPayment
	return d1, sel
}

func TestHandleDealInvoiceExpired_usesInjectedClock(t *testing.T) {
	clk := expFixedClock{t: time.Date(2026, 7, 15, 14, 30, 0, 0, time.UTC)}
	d1, sel := paymentTimeoutDealTwoWinners(t)
	deals := &mapDealRepo{byID: map[string]*deal.Deal{d1.ID(): d1}}
	selections := &mapSelectionRepo{byAuction: map[string]*deal.WinnerSelection{sel.AuctionID: sel}}
	out := &sliceOutbox{}
	uow := NewSimpleUnitOfWork(deals, &confirmationRepoSpy{}, &projectionRepoSpy{}, selections, out)
	uc, err := NewHandleDealInvoiceExpired(uow, clk)
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.Execute(context.Background(), wallet.DealInvoiceExpired{
		DealID:         d1.ID(),
		AuctionID:      d1.AuctionID(),
		BuyerCompanyID: d1.CustomerID(),
	}); err != nil {
		t.Fatal(err)
	}
	for _, e := range out.saved[0] {
		if nw, ok := e.(deal.NextWinnerSelected); ok {
			if !nw.SelectedAt.Equal(clk.t) {
				t.Fatalf("NextWinnerSelected.SelectedAt = %v, want %v", nw.SelectedAt, clk.t)
			}
			return
		}
	}
	t.Fatal("expected NextWinnerSelected in batch")
}

func TestHandleDealInvoiceExpired_movesToNextWinner(t *testing.T) {
	d1, sel := paymentTimeoutDealTwoWinners(t)
	deals := &mapDealRepo{byID: map[string]*deal.Deal{d1.ID(): d1}}
	selections := &mapSelectionRepo{byAuction: map[string]*deal.WinnerSelection{sel.AuctionID: sel}}
	out := &sliceOutbox{}
	uow := NewSimpleUnitOfWork(deals, &confirmationRepoSpy{}, &projectionRepoSpy{}, selections, out)

	uc, err := NewHandleDealInvoiceExpired(uow, nil)
	if err != nil {
		t.Fatal(err)
	}
	evt := wallet.DealInvoiceExpired{
		InvoiceID:      "inv-1",
		DealID:         d1.ID(),
		AuctionID:      d1.AuctionID(),
		BuyerCompanyID: d1.CustomerID(),
		Amount:         100,
		Currency:       wallet.CurrencyRUB,
		ExpiredAt:      time.Now().UTC(),
	}
	if err := uc.Execute(context.Background(), evt); err != nil {
		t.Fatal(err)
	}
	if d1.Status() != deal.DealStatusCancelled {
		t.Fatalf("d1 cancelled: %s", d1.Status())
	}
	d2 := deals.byID[sel.DealID]
	if d2 == nil || d2.CustomerID() != "buyer-2" {
		t.Fatalf("expected new deal for buyer-2, sel.DealID=%q deals=%v", sel.DealID, deals.byID)
	}
	if len(out.saved) != 1 {
		t.Fatalf("batches: %d", len(out.saved))
	}
	batch := out.saved[0]
	var nCancel, nWR, nCreated, nNext int
	for _, e := range batch {
		switch e.(type) {
		case deal.DealCancelled:
			nCancel++
		case deal.WinnerRejected:
			nWR++
		case deal.DealCreated:
			nCreated++
		case deal.NextWinnerSelected:
			nNext++
		}
	}
	if nCancel != 1 || nWR != 1 || nCreated != 1 || nNext != 1 {
		t.Fatalf("event counts cancel=%d wr=%d created=%d next=%d batch=%#v", nCancel, nWR, nCreated, nNext, batch)
	}
}

func TestHandleDealInvoiceExpired_exhaustedSingleCandidate(t *testing.T) {
	d1, sel := paymentTimeoutDealTwoWinners(t)
	sel.Candidates = []string{"buyer-1"}
	deals := &mapDealRepo{byID: map[string]*deal.Deal{d1.ID(): d1}}
	selections := &mapSelectionRepo{byAuction: map[string]*deal.WinnerSelection{sel.AuctionID: sel}}
	out := &sliceOutbox{}
	uow := NewSimpleUnitOfWork(deals, &confirmationRepoSpy{}, &projectionRepoSpy{}, selections, out)
	uc, err := NewHandleDealInvoiceExpired(uow, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.Execute(context.Background(), wallet.DealInvoiceExpired{
		DealID:         d1.ID(),
		AuctionID:      d1.AuctionID(),
		BuyerCompanyID: d1.CustomerID(),
		ExpiredAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if sel.Status != deal.WinnerSelectionExhausted {
		t.Fatalf("selection: %s", sel.Status)
	}
	batch := out.saved[0]
	var nFail int
	for _, e := range batch {
		if _, ok := e.(deal.WinnerSelectionFailed); ok {
			nFail++
		}
	}
	if nFail != 1 {
		t.Fatalf("expected WinnerSelectionFailed: %#v", batch)
	}
	var nWR int
	for _, e := range batch {
		if _, ok := e.(deal.WinnerRejected); ok {
			nWR++
		}
	}
	if nWR != 1 {
		t.Fatalf("expected single WinnerRejected, got %d", nWR)
	}
}

func TestHandleDealInvoiceExpired_replaySelectionPointsToOtherDeal(t *testing.T) {
	d1, sel := paymentTimeoutDealTwoWinners(t)
	d2, _, err := deal.NewFactory().CreateFromSelection(sel.AuctionID, sel.SupplierID, sel.ProductSnapshot, "buyer-2", sel.FinalPrice, sel.WonAt)
	if err != nil {
		t.Fatal(err)
	}
	sel.DealID = d2.ID()
	sel.Status = deal.WinnerSelectionActive
	sel.CurrentIndex = 1
	deals := &mapDealRepo{byID: map[string]*deal.Deal{d1.ID(): d1, d2.ID(): d2}}
	selections := &mapSelectionRepo{byAuction: map[string]*deal.WinnerSelection{sel.AuctionID: sel}}
	out := &sliceOutbox{}
	uow := NewSimpleUnitOfWork(deals, &confirmationRepoSpy{}, &projectionRepoSpy{}, selections, out)
	uc, err := NewHandleDealInvoiceExpired(uow, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.Execute(context.Background(), wallet.DealInvoiceExpired{
		DealID:         d1.ID(),
		AuctionID:      d1.AuctionID(),
		BuyerCompanyID: d1.CustomerID(),
		ExpiredAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if len(out.saved) != 0 {
		t.Fatalf("expected no outbox, got %v", out.saved)
	}
}

func TestHandleDealInvoiceExpired_cancelledDealNoOp(t *testing.T) {
	d1, sel := paymentTimeoutDealTwoWinners(t)
	if _, err := d1.Cancel("x", "system"); err != nil {
		t.Fatal(err)
	}
	deals := &mapDealRepo{byID: map[string]*deal.Deal{d1.ID(): d1}}
	selections := &mapSelectionRepo{byAuction: map[string]*deal.WinnerSelection{sel.AuctionID: sel}}
	out := &sliceOutbox{}
	uow := NewSimpleUnitOfWork(deals, &confirmationRepoSpy{}, &projectionRepoSpy{}, selections, out)
	uc, err := NewHandleDealInvoiceExpired(uow, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.Execute(context.Background(), wallet.DealInvoiceExpired{
		DealID:         d1.ID(),
		AuctionID:      d1.AuctionID(),
		BuyerCompanyID: d1.CustomerID(),
		ExpiredAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if len(out.saved) != 0 {
		t.Fatal("expected no events")
	}
}
