package app

import (
	"context"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type memDealInvoiceRepo struct {
	byID   map[string]*wallet.DealInvoice
	byDeal map[string]*wallet.DealInvoice
}

func newMemDealInvoiceRepo() *memDealInvoiceRepo {
	return &memDealInvoiceRepo{byID: make(map[string]*wallet.DealInvoice), byDeal: make(map[string]*wallet.DealInvoice)}
}

func (m *memDealInvoiceRepo) Create(ctx context.Context, inv *wallet.DealInvoice) error {
	_ = ctx
	m.byID[inv.ID] = inv
	m.byDeal[inv.DealID] = inv
	return nil
}

func (m *memDealInvoiceRepo) Save(ctx context.Context, inv *wallet.DealInvoice) error {
	_ = ctx
	m.byID[inv.ID] = inv
	m.byDeal[inv.DealID] = inv
	return nil
}

func (m *memDealInvoiceRepo) LoadByDealID(ctx context.Context, dealID string) (*wallet.DealInvoice, error) {
	_ = ctx
	inv, ok := m.byDeal[dealID]
	if !ok {
		return nil, ErrDealInvoiceNotFound
	}
	return inv, nil
}

func (m *memDealInvoiceRepo) LoadByDealIDForUpdate(ctx context.Context, dealID string) (*wallet.DealInvoice, error) {
	return m.LoadByDealID(ctx, dealID)
}

func (m *memDealInvoiceRepo) LoadByID(ctx context.Context, id string) (*wallet.DealInvoice, error) {
	_ = ctx
	inv, ok := m.byID[id]
	if !ok {
		return nil, ErrDealInvoiceNotFound
	}
	return inv, nil
}

func (m *memDealInvoiceRepo) LoadByIDForUpdate(ctx context.Context, id string) (*wallet.DealInvoice, error) {
	return m.LoadByID(ctx, id)
}

func (m *memDealInvoiceRepo) ListByBuyerCompany(ctx context.Context, buyerCompanyID string, limit int) ([]*wallet.DealInvoice, error) {
	_ = ctx
	_ = limit
	var out []*wallet.DealInvoice
	for _, inv := range m.byID {
		if inv.BuyerCompanyID == buyerCompanyID {
			out = append(out, inv)
		}
	}
	return out, nil
}

type capturePublisher struct {
	events []any
}

func (c *capturePublisher) Publish(ctx context.Context, aggregateID, companyID string, event any) error {
	_ = ctx
	_ = aggregateID
	_ = companyID
	c.events = append(c.events, event)
	return nil
}

type countingDealStub struct {
	stubFakeProvider
	calls int
}

func (c *countingDealStub) CreateDealInvoice(ctx context.Context, req CreateDealInvoiceRequest) (CreateDealInvoiceResponse, error) {
	c.calls++
	return c.stubFakeProvider.CreateDealInvoice(ctx, req)
}

func TestCreateDealInvoice_IdempotentSecondCallSkipsProvider(t *testing.T) {
	ctx := context.Background()
	repo := newMemDealInvoiceRepo()
	deps := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{}}
	prov := &countingDealStub{}
	pub := &capturePublisher{}
	uc, err := NewCreateDealInvoice(repo, deps, prov, stubFakeProviderName, RandomHexID{}, nil, pub, "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	cmd := CreateDealInvoiceCommand{
		DealID:          "d1",
		AuctionID:       "",
		BuyerCompanyID:  "b1",
		SellerCompanyID: "s1",
		GoodsAmount:     1000,
		Currency:        wallet.CurrencyRUB,
	}
	inv1, err := uc.Execute(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	inv2, err := uc.Execute(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if inv1.ID != inv2.ID {
		t.Fatalf("expected same invoice, got %s vs %s", inv1.ID, inv2.ID)
	}
	if prov.calls != 1 {
		t.Fatalf("provider calls: want 1 got %d", prov.calls)
	}
	var created int
	for _, e := range pub.events {
		if _, ok := e.(wallet.DealInvoiceCreated); ok {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("DealInvoiceCreated events: want 1 got %d", created)
	}
}

func TestCreateDealInvoice_PlatformFeeDueNetOfHeldDeposit(t *testing.T) {
	ctx := context.Background()
	repo := newMemDealInvoiceRepo()
	deps := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{
		depKey("a1", "b1"): {
			AuctionID: "a1",
			CompanyID: "b1",
			Amount:    10_000,
			Status:    wallet.DepositHeld,
		},
	}}
	uc, err := NewCreateDealInvoice(repo, deps, stubFakeProvider{}, stubFakeProviderName, RandomHexID{}, nil, nil, "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	goods := int64(1_000_000)
	wantFeePart := platformFeeFromFinalPrice(goods) - 10_000
	if wantFeePart < 0 {
		wantFeePart = 0
	}
	inv, err := uc.Execute(ctx, CreateDealInvoiceCommand{
		DealID:          "d-fee",
		AuctionID:       "a1",
		BuyerCompanyID:  "b1",
		SellerCompanyID: "s1",
		GoodsAmount:     goods,
		Currency:        wallet.CurrencyRUB,
	})
	if err != nil {
		t.Fatal(err)
	}
	if inv.PlatformFeeDueAmount != wantFeePart {
		t.Fatalf("PlatformFeeDueAmount: want %d got %d", wantFeePart, inv.PlatformFeeDueAmount)
	}
	if inv.TotalAmount != goods+wantFeePart {
		t.Fatalf("TotalAmount: want %d got %d", goods+wantFeePart, inv.TotalAmount)
	}
}

func TestConfirmDealInvoicePaid_IdempotentNoExtraEvent(t *testing.T) {
	ctx := context.Background()
	repo := newMemDealInvoiceRepo()
	now := time.Date(2026, 1, 2, 15, 0, 0, 0, time.UTC)
	inv, err := wallet.NewDealInvoice("inv1", "d1", "", "b1", "s1", 100, 5, wallet.CurrencyRUB, stubFakeProviderName, now.Add(24*time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.AttachProvider("pid", "http://pay"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatal(err)
	}
	pub := &capturePublisher{}
	uc, err := NewConfirmDealInvoicePaid(repo, pub, fixedClock{t: now})
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.Execute(ctx, "inv1"); err != nil {
		t.Fatal(err)
	}
	if err := uc.Execute(ctx, "inv1"); err != nil {
		t.Fatal(err)
	}
	var paid int
	for _, e := range pub.events {
		if _, ok := e.(wallet.DealInvoicePaid); ok {
			paid++
		}
	}
	if paid != 1 {
		t.Fatalf("DealInvoicePaid events: want 1 got %d", paid)
	}
}
