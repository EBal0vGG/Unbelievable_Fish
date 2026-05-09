package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type memSellerPayoutRepo struct {
	byDeal map[string]*wallet.SellerPayout
	byID   map[string]*wallet.SellerPayout
}

func newMemSellerPayoutRepo() *memSellerPayoutRepo {
	return &memSellerPayoutRepo{
		byDeal: make(map[string]*wallet.SellerPayout),
		byID:   make(map[string]*wallet.SellerPayout),
	}
}

func (m *memSellerPayoutRepo) Create(_ context.Context, p *wallet.SellerPayout) error {
	m.byDeal[p.DealID] = p
	m.byID[p.ID] = p
	return nil
}

func (m *memSellerPayoutRepo) Save(_ context.Context, p *wallet.SellerPayout) error {
	m.byDeal[p.DealID] = p
	m.byID[p.ID] = p
	return nil
}

func (m *memSellerPayoutRepo) LoadByID(_ context.Context, id string) (*wallet.SellerPayout, error) {
	p, ok := m.byID[id]
	if !ok {
		return nil, ErrSellerPayoutNotFound
	}
	return p, nil
}

func (m *memSellerPayoutRepo) LoadByIDForUpdate(ctx context.Context, id string) (*wallet.SellerPayout, error) {
	return m.LoadByID(ctx, id)
}

func (m *memSellerPayoutRepo) LoadByDealID(_ context.Context, dealID string) (*wallet.SellerPayout, error) {
	return m.loadDeal(dealID)
}

func (m *memSellerPayoutRepo) LoadByDealIDForUpdate(_ context.Context, dealID string) (*wallet.SellerPayout, error) {
	return m.loadDeal(dealID)
}

func (m *memSellerPayoutRepo) loadDeal(dealID string) (*wallet.SellerPayout, error) {
	p, ok := m.byDeal[dealID]
	if !ok {
		return nil, ErrSellerPayoutNotFound
	}
	return p, nil
}

func (m *memSellerPayoutRepo) ListBySellerCompany(_ context.Context, sellerCompanyID string, limit int) ([]*wallet.SellerPayout, error) {
	if limit <= 0 {
		limit = 50
	}
	var out []*wallet.SellerPayout
	for _, p := range m.byDeal {
		if p.SellerCompanyID == sellerCompanyID {
			out = append(out, p)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func TestCreateSellerPayout_happyPath_goodsAmountPending(t *testing.T) {
	ctx := context.Background()
	invRepo := newMemDealInvoiceRepo()
	payoutRepo := newMemSellerPayoutRepo()
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	inv, err := wallet.NewDealInvoice("inv-p1", "deal-p1", "auc-p1", "buyer-p1", "seller-p1", 100_000, 2_000, wallet.CurrencyRUB, stubFakeProviderName, now.Add(time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.AttachProvider("p1", "http://pay"); err != nil {
		t.Fatal(err)
	}
	if err := inv.MarkPaid(inv.TotalAmount, inv.Currency, now); err != nil {
		t.Fatal(err)
	}
	if err := invRepo.Create(ctx, inv); err != nil {
		t.Fatal(err)
	}
	pub := &capturePublisher{}
	uc, err := NewCreateSellerPayout(payoutRepo, invRepo, RandomHexID{}, fixedClock{t: now}, pub)
	if err != nil {
		t.Fatal(err)
	}
	p, err := uc.Execute(ctx, CreateSellerPayoutCommand{
		DealID:               "deal-p1",
		AuctionID:            "auc-p1",
		BuyerCompanyID:       "buyer-p1",
		GoodsAmount:          100_000,
		PlatformFeeDueAmount: 2_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Amount != 100_000 || p.Status != wallet.SellerPayoutPending {
		t.Fatalf("payout: %+v", p)
	}
	if len(pub.events) != 1 {
		t.Fatalf("events: %d", len(pub.events))
	}
	ev, ok := pub.events[0].(wallet.SellerPayoutCreated)
	if !ok || ev.Amount != 100_000 {
		t.Fatalf("event: %+v", pub.events[0])
	}
}

func TestCreateSellerPayout_idempotentSecondCall(t *testing.T) {
	ctx := context.Background()
	invRepo := newMemDealInvoiceRepo()
	payoutRepo := newMemSellerPayoutRepo()
	now := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	inv, err := wallet.NewDealInvoice("inv-p2", "deal-p2", "auc-x", "b1", "s1", 50, 5, wallet.CurrencyRUB, stubFakeProviderName, now.Add(time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.AttachProvider("p1", "http://pay"); err != nil {
		t.Fatal(err)
	}
	if err := inv.MarkPaid(inv.TotalAmount, inv.Currency, now); err != nil {
		t.Fatal(err)
	}
	if err := invRepo.Create(ctx, inv); err != nil {
		t.Fatal(err)
	}
	pub := &capturePublisher{}
	uc, err := NewCreateSellerPayout(payoutRepo, invRepo, RandomHexID{}, fixedClock{t: now}, pub)
	if err != nil {
		t.Fatal(err)
	}
	cmd := CreateSellerPayoutCommand{DealID: "deal-p2", AuctionID: "auc-x", BuyerCompanyID: "b1", GoodsAmount: 50, PlatformFeeDueAmount: 5}
	p1, err := uc.Execute(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := uc.Execute(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if p1.ID != p2.ID {
		t.Fatal("expected same payout id")
	}
	if len(pub.events) != 1 {
		t.Fatalf("want one SellerPayoutCreated, got %d", len(pub.events))
	}
}

func TestCreateSellerPayout_usesGoodsNotTotal(t *testing.T) {
	ctx := context.Background()
	invRepo := newMemDealInvoiceRepo()
	payoutRepo := newMemSellerPayoutRepo()
	now := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	inv, err := wallet.NewDealInvoice("inv-p3", "deal-p3", "auc-x", "b1", "s1", 100_000, 2_000, wallet.CurrencyRUB, stubFakeProviderName, now.Add(time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.AttachProvider("p1", "http://pay"); err != nil {
		t.Fatal(err)
	}
	if err := inv.MarkPaid(inv.TotalAmount, inv.Currency, now); err != nil {
		t.Fatal(err)
	}
	if inv.TotalAmount != 102_000 {
		t.Fatalf("total: %d", inv.TotalAmount)
	}
	if err := invRepo.Create(ctx, inv); err != nil {
		t.Fatal(err)
	}
	pub := &capturePublisher{}
	uc, err := NewCreateSellerPayout(payoutRepo, invRepo, RandomHexID{}, fixedClock{t: now}, pub)
	if err != nil {
		t.Fatal(err)
	}
	p, err := uc.Execute(ctx, CreateSellerPayoutCommand{
		DealID: "deal-p3", AuctionID: "auc-x", BuyerCompanyID: "b1", GoodsAmount: 100_000, PlatformFeeDueAmount: 2_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Amount != 100_000 {
		t.Fatalf("payout amount: %d", p.Amount)
	}
	_ = pub
}

func TestCreateSellerPayout_buyerEqualsSellerRejected(t *testing.T) {
	ctx := context.Background()
	invRepo := newMemDealInvoiceRepo()
	payoutRepo := newMemSellerPayoutRepo()
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	inv, err := wallet.NewDealInvoice("inv-bad", "deal-bad", "auc-x", "same", "other", 10, 0, wallet.CurrencyRUB, stubFakeProviderName, now.Add(time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	// Corrupt snapshot: same buyer and seller (invalid for payout)
	inv.BuyerCompanyID = "same"
	inv.SellerCompanyID = "same"
	if err := inv.AttachProvider("p1", "http://pay"); err != nil {
		t.Fatal(err)
	}
	if err := inv.MarkPaid(inv.TotalAmount, inv.Currency, now); err != nil {
		t.Fatal(err)
	}
	if err := invRepo.Create(ctx, inv); err != nil {
		t.Fatal(err)
	}
	uc, err := NewCreateSellerPayout(payoutRepo, invRepo, RandomHexID{}, fixedClock{t: now}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.Execute(ctx, CreateSellerPayoutCommand{
		DealID: "deal-bad", AuctionID: "auc-x", BuyerCompanyID: "same", GoodsAmount: 10, PlatformFeeDueAmount: 0,
	})
	if !errors.Is(err, wallet.ErrInvalidSellerPayout) {
		t.Fatalf("want ErrInvalidSellerPayout got %v", err)
	}
}

func TestCreateSellerPayout_invoiceNotPaid(t *testing.T) {
	ctx := context.Background()
	invRepo := newMemDealInvoiceRepo()
	payoutRepo := newMemSellerPayoutRepo()
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	inv, err := wallet.NewDealInvoice("inv-np", "deal-np", "auc-x", "b1", "s1", 10, 0, wallet.CurrencyRUB, stubFakeProviderName, now.Add(time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.AttachProvider("p1", "http://pay"); err != nil {
		t.Fatal(err)
	}
	if err := invRepo.Create(ctx, inv); err != nil {
		t.Fatal(err)
	}
	uc, err := NewCreateSellerPayout(payoutRepo, invRepo, RandomHexID{}, fixedClock{t: now}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.Execute(ctx, CreateSellerPayoutCommand{
		DealID: "deal-np", AuctionID: "auc-x", BuyerCompanyID: "b1", GoodsAmount: 10, PlatformFeeDueAmount: 0,
	})
	if !errors.Is(err, wallet.ErrInvoiceNotPayable) {
		t.Fatalf("want ErrInvoiceNotPayable got %v", err)
	}
}
