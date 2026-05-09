package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

func TestMarkSellerPayoutReady_pendingToReady(t *testing.T) {
	ctx := context.Background()
	repo := newMemSellerPayoutRepo()
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	p, err := wallet.NewSellerPayout("po1", "d1", "inv1", "a1", "seller1", "buyer1", 50_000, wallet.CurrencyRUB, wallet.SellerPayoutPending, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	pub := &capturePublisher{}
	uc, err := NewMarkSellerPayoutReady(repo, fixedClock{t: now}, pub)
	if err != nil {
		t.Fatal(err)
	}
	out, err := uc.Execute(ctx, "po1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != wallet.SellerPayoutReady {
		t.Fatalf("status %s", out.Status)
	}
	if len(pub.events) != 1 {
		t.Fatalf("events %d", len(pub.events))
	}
	if _, ok := pub.events[0].(wallet.SellerPayoutMarkedReady); !ok {
		t.Fatalf("event type %T", pub.events[0])
	}
	out2, err := uc.Execute(ctx, "po1")
	if err != nil {
		t.Fatal(err)
	}
	if out2.Status != wallet.SellerPayoutReady {
		t.Fatal(out2.Status)
	}
	if len(pub.events) != 1 {
		t.Fatalf("idempotent should not publish again, got %d", len(pub.events))
	}
}

func TestMarkSellerPayoutReady_wrongStatus(t *testing.T) {
	ctx := context.Background()
	repo := newMemSellerPayoutRepo()
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	p, err := wallet.NewSellerPayout("po2", "d2", "inv2", "a2", "seller2", "buyer2", 1, wallet.CurrencyRUB, wallet.SellerPayoutCancelled, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	uc, err := NewMarkSellerPayoutReady(repo, fixedClock{t: now}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.Execute(ctx, "po2")
	if !errors.Is(err, wallet.ErrSellerPayoutWrongStatus) {
		t.Fatalf("got %v", err)
	}
}

func TestMarkSellerPayoutPaid_readyCreditsSeller(t *testing.T) {
	ctx := context.Background()
	payoutRepo := newMemSellerPayoutRepo()
	ar := &memAccountRepo{accounts: make(map[string]*wallet.Account)}
	ledger := &memLedgerRepo{}
	createAccount, err := NewCreateAccount(ar, RandomHexID{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	p, err := wallet.NewSellerPayout("po3", "d3", "inv3", "a3", "seller3", "buyer3", 80_000, wallet.CurrencyRUB, wallet.SellerPayoutPending, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.MarkReady(now); err != nil {
		t.Fatal(err)
	}
	if err := payoutRepo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	pub := &capturePublisher{}
	paidUC, err := NewMarkSellerPayoutPaid(payoutRepo, ar, ledger, createAccount, RandomHexID{}, fixedClock{t: now}, pub)
	if err != nil {
		t.Fatal(err)
	}
	out, err := paidUC.Execute(ctx, "po3")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != wallet.SellerPayoutPaid {
		t.Fatalf("status %s", out.Status)
	}
	acc, err := ar.LoadByCompany(ctx, "seller3")
	if err != nil {
		t.Fatal(err)
	}
	if acc.Available() != 80_000 {
		t.Fatalf("available %d", acc.Available())
	}
	if len(ledger.entries) != 1 {
		t.Fatalf("ledger %d", len(ledger.entries))
	}
	if ledger.entries[0].EntryType != wallet.LedgerSellerPayoutCredited {
		t.Fatal(ledger.entries[0].EntryType)
	}
	if len(pub.events) != 1 {
		t.Fatalf("events %d", len(pub.events))
	}
	if _, ok := pub.events[0].(wallet.SellerPayoutMarkedPaid); !ok {
		t.Fatalf("event %T", pub.events[0])
	}
	outAgain, err := paidUC.Execute(ctx, "po3")
	if err != nil {
		t.Fatal(err)
	}
	if outAgain.Status != wallet.SellerPayoutPaid {
		t.Fatal(outAgain.Status)
	}
	if acc2, _ := ar.LoadByCompany(ctx, "seller3"); acc2.Available() != 80_000 {
		t.Fatal("double credit")
	}
	if len(ledger.entries) != 1 {
		t.Fatalf("ledger after idempotent: %d", len(ledger.entries))
	}
	if len(pub.events) != 1 {
		t.Fatalf("events after idempotent: %d", len(pub.events))
	}
}

func TestMarkSellerPayoutPaid_readyWithExistingLedgerNoSecondDeposit(t *testing.T) {
	ctx := context.Background()
	payoutRepo := newMemSellerPayoutRepo()
	ar := &memAccountRepo{accounts: make(map[string]*wallet.Account)}
	ledger := &memLedgerRepo{}
	createAccount, err := NewCreateAccount(ar, RandomHexID{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	acc, err := wallet.RehydrateAccount("acc-s5", "seller5", wallet.CurrencyRUB, 50_000, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := ar.Create(ctx, acc); err != nil {
		t.Fatal(err)
	}
	p, err := wallet.NewSellerPayout("po5", "d5", "inv5", "a5", "seller5", "buyer5", 50_000, wallet.CurrencyRUB, wallet.SellerPayoutPending, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.MarkReady(now); err != nil {
		t.Fatal(err)
	}
	if err := payoutRepo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Append(ctx, wallet.LedgerEntry{
		ID:            "pre-led",
		AccountID:     acc.ID(),
		CompanyID:     "seller5",
		Currency:      wallet.CurrencyRUB,
		Amount:        50_000,
		EntryType:     wallet.LedgerSellerPayoutCredited,
		ReferenceType: "seller_payout",
		ReferenceID:   "po5",
		Reason:        "SELLER_PAYOUT_PAID",
		CreatedAt:     now,
	}); err != nil {
		t.Fatal(err)
	}
	pub := &capturePublisher{}
	paidUC, err := NewMarkSellerPayoutPaid(payoutRepo, ar, ledger, createAccount, RandomHexID{}, fixedClock{t: now}, pub)
	if err != nil {
		t.Fatal(err)
	}
	out, err := paidUC.Execute(ctx, "po5")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != wallet.SellerPayoutPaid {
		t.Fatalf("status %s", out.Status)
	}
	acc2, err := ar.LoadByCompany(ctx, "seller5")
	if err != nil {
		t.Fatal(err)
	}
	if acc2.Available() != 50_000 {
		t.Fatalf("no second deposit: want 50000 got %d", acc2.Available())
	}
	if len(ledger.entries) != 1 {
		t.Fatalf("ledger rows: %d", len(ledger.entries))
	}
}

func TestMarkSellerPayoutPaid_pendingErrors(t *testing.T) {
	ctx := context.Background()
	payoutRepo := newMemSellerPayoutRepo()
	ar := &memAccountRepo{accounts: make(map[string]*wallet.Account)}
	ledger := &memLedgerRepo{}
	createAccount, err := NewCreateAccount(ar, RandomHexID{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	p, err := wallet.NewSellerPayout("po4", "d4", "inv4", "a4", "seller4", "buyer4", 10, wallet.CurrencyRUB, wallet.SellerPayoutPending, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := payoutRepo.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	paidUC, err := NewMarkSellerPayoutPaid(payoutRepo, ar, ledger, createAccount, RandomHexID{}, fixedClock{t: now}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = paidUC.Execute(ctx, "po4")
	if !errors.Is(err, wallet.ErrSellerPayoutWrongStatus) {
		t.Fatalf("got %v", err)
	}
}
