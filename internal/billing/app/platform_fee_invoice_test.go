package app

import (
	"context"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

// Сценарии A–C: снимок platform_fee_due и total при создании инвойса (3% от goods).
// goods = 100_000 → fee = 3_000 (целочисленное деление как в platformFeeFromFinalPrice).

func TestCreateDealInvoice_PlatformFeeDue_CaseA_DepositExceedsFee(t *testing.T) {
	const goods = int64(100_000)
	const held = int64(5_000) // > fee 3_000
	testCreateDealInvoiceFeeScenario(t, goods, held, 0, goods, "A")
}

func TestCreateDealInvoice_PlatformFeeDue_CaseB_DepositEqualsFee(t *testing.T) {
	const goods = int64(100_000)
	const held = int64(3_000) // == fee
	testCreateDealInvoiceFeeScenario(t, goods, held, 0, goods, "B")
}

func TestCreateDealInvoice_PlatformFeeDue_CaseC_DepositLessThanFee(t *testing.T) {
	const goods = int64(100_000)
	const held = int64(1_000)
	const wantFeeDue = int64(2_000) // 3000 - 1000
	testCreateDealInvoiceFeeScenario(t, goods, held, wantFeeDue, goods+wantFeeDue, "C")
}

func testCreateDealInvoiceFeeScenario(t *testing.T, goods, held, wantFeeDue, wantTotal int64, name string) {
	t.Helper()
	ctx := context.Background()
	repo := newMemDealInvoiceRepo()
	deps := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{
		depKey("auc-"+name, "buyer-"+name): {
			AuctionID: "auc-" + name,
			CompanyID: "buyer-" + name,
			Amount:    held,
			Status:    wallet.DepositHeld,
		},
	}}
	uc, err := NewCreateDealInvoice(repo, deps, stubFakeProvider{}, stubFakeProviderName, RandomHexID{}, nil, nil, "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := uc.Execute(ctx, CreateDealInvoiceCommand{
		DealID:          "deal-" + name,
		AuctionID:       "auc-" + name,
		BuyerCompanyID:  "buyer-" + name,
		SellerCompanyID: "seller-" + name,
		GoodsAmount:     goods,
		Currency:        wallet.CurrencyRUB,
	})
	if err != nil {
		t.Fatal(err)
	}
	if inv.PlatformFeeDueAmount != wantFeeDue {
		t.Fatalf("%s: PlatformFeeDueAmount want %d got %d", name, wantFeeDue, inv.PlatformFeeDueAmount)
	}
	if inv.TotalAmount != wantTotal {
		t.Fatalf("%s: TotalAmount want %d got %d", name, wantTotal, inv.TotalAmount)
	}
	feeFull := platformFeeFromFinalPrice(goods)
	if feeFull != 3_000 {
		t.Fatalf("%s: platformFeeFromFinalPrice(%d) want 3000 got %d", name, goods, feeFull)
	}
}

// TestConfirmDealInvoicePaid_Idempotent_NoExtraDomainEvents: повторный confirm не дублирует DealInvoicePaid (Stage 9: ledger не трогается).
func TestConfirmDealInvoicePaid_Idempotent_NoExtraDomainEvents(t *testing.T) {
	ctx := context.Background()
	repo := newMemDealInvoiceRepo()
	clk := fixedClock{t: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	inv, err := wallet.NewDealInvoice("inv-d", "d-d", "", "b-d", "s-d", 100_000, 2_000, wallet.CurrencyRUB, stubFakeProviderName, clk.Now().Add(24*time.Hour), clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.AttachProvider("pid-d", "http://localhost/billing/invoices/inv-d/fake-confirm"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, inv); err != nil {
		t.Fatal(err)
	}
	pub := &capturePublisher{}
	uc, err := NewConfirmDealInvoicePaid(repo, pub, clk)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := uc.Execute(ctx, "inv-d"); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
	var paid int
	for _, e := range pub.events {
		if _, ok := e.(wallet.DealInvoicePaid); ok {
			paid++
		}
	}
	if paid != 1 {
		t.Fatalf("DealInvoicePaid published: want 1 got %d", paid)
	}
}

// Stage 10+: полное settlement (ledger capture/release/platform revenue, идемпотентность) — см. orchestrator DealInvoicePaid → FinalizeDealPayment.
func TestStage10_Settlement_Ledger_scenarios_skipped(t *testing.T) {
	t.Skip("CASE A–D ledger/release/capture: реализовать вместе с FinalizeDealPayment; ожидания: A) fee_due=0, затем revenue 3000 + release 2000; B) fee_due=0, capture 3000; C) fee_due=2000, capture held 1000; D) повтор без дублей в ledger")
}
