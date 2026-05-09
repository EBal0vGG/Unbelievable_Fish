package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

func TestSettleWinnerDepositAfterInvoicePaid_partialCaptureAndRelease_settled(t *testing.T) {
	ctx := context.Background()
	clk := fixedClock{t: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	const goods int64 = 100_000
	fee := platformFeeFromFinalPrice(goods)
	if fee != 3000 {
		t.Fatalf("fee sanity: want 3000 got %d", fee)
	}
	const held int64 = 10_000

	ar, deps, ledger, uc := newSettleWinnerFixture(clk)
	seedHeldDeposit(t, ar, deps, "auc1", "co1", held, clk.Now())

	if err := uc.Execute(ctx, "auc1", "co1", goods, 0, "WINNER_FINALIZED"); err != nil {
		t.Fatal(err)
	}
	dep := deps.deposits[depKey("auc1", "co1")]
	if dep.Status != wallet.DepositSettled {
		t.Fatalf("deposit status: want SETTLED got %s", dep.Status)
	}
	var captured, released int
	for _, e := range ledger.entries {
		switch e.EntryType {
		case wallet.LedgerPlatformFeeCaptured:
			captured++
			if e.Amount != fee {
				t.Fatalf("fee captured amount: want %d got %d", fee, e.Amount)
			}
		case wallet.LedgerBidDepositReleased:
			released++
			if e.Amount != held-fee {
				t.Fatalf("release amount: want %d got %d", held-fee, e.Amount)
			}
		case wallet.LedgerPlatformFeeDue:
			t.Fatalf("unexpected PLATFORM_FEE_DUE ledger row")
		}
	}
	if captured != 1 || released != 1 {
		t.Fatalf("ledger rows: want 1 fee + 1 release, got entries=%+v", ledger.entries)
	}
}

func TestSettleWinnerDepositAfterInvoicePaid_fullCapture_captured(t *testing.T) {
	ctx := context.Background()
	clk := fixedClock{t: time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)}
	const goods int64 = 100_000
	fee := platformFeeFromFinalPrice(goods)

	ar, deps, ledger, uc := newSettleWinnerFixture(clk)
	seedHeldDeposit(t, ar, deps, "auc2", "co2", fee, clk.Now())

	if err := uc.Execute(ctx, "auc2", "co2", goods, 0, "WINNER_FINALIZED"); err != nil {
		t.Fatal(err)
	}
	if deps.deposits[depKey("auc2", "co2")].Status != wallet.DepositCaptured {
		t.Fatalf("want CAPTURED got %s", deps.deposits[depKey("auc2", "co2")].Status)
	}
	if len(ledger.entries) != 1 || ledger.entries[0].EntryType != wallet.LedgerPlatformFeeCaptured {
		t.Fatalf("ledger: %+v", ledger.entries)
	}
}

func TestSettleWinnerDepositAfterInvoicePaid_depositSmallerThanFee_usesInvoiceDue(t *testing.T) {
	ctx := context.Background()
	clk := fixedClock{t: time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)}
	const goods int64 = 100_000
	fee := platformFeeFromFinalPrice(goods)
	const held int64 = 1000
	wantDue := fee - held

	ar, deps, ledger, uc := newSettleWinnerFixture(clk)
	seedHeldDeposit(t, ar, deps, "auc3", "co3", held, clk.Now())

	if err := uc.Execute(ctx, "auc3", "co3", goods, wantDue, "WINNER_FINALIZED"); err != nil {
		t.Fatal(err)
	}
	if deps.deposits[depKey("auc3", "co3")].Status != wallet.DepositCaptured {
		t.Fatalf("want CAPTURED got %s", deps.deposits[depKey("auc3", "co3")].Status)
	}
	if len(ledger.entries) != 1 || ledger.entries[0].Amount != held {
		t.Fatalf("ledger: %+v", ledger.entries)
	}
}

func TestSettleWinnerDepositAfterInvoicePaid_zeroFee_releasesDeposit(t *testing.T) {
	ctx := context.Background()
	clk := fixedClock{t: time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)}
	const goods int64 = 30 // 30*3/100 = 0
	if platformFeeFromFinalPrice(goods) != 0 {
		t.Fatal("test expects zero platform fee")
	}
	const held int64 = 500

	ar, deps, ledger, uc := newSettleWinnerFixture(clk)
	seedHeldDeposit(t, ar, deps, "auc4", "co4", held, clk.Now())

	if err := uc.Execute(ctx, "auc4", "co4", goods, 0, "WINNER_FINALIZED"); err != nil {
		t.Fatal(err)
	}
	if deps.deposits[depKey("auc4", "co4")].Status != wallet.DepositReleased {
		t.Fatalf("want RELEASED got %s", deps.deposits[depKey("auc4", "co4")].Status)
	}
	if len(ledger.entries) != 1 || ledger.entries[0].EntryType != wallet.LedgerBidDepositReleased {
		t.Fatalf("ledger: %+v", ledger.entries)
	}
}

func TestSettleWinnerDepositAfterInvoicePaid_platformFeeDueMismatch(t *testing.T) {
	ctx := context.Background()
	clk := fixedClock{t: time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)}
	const goods int64 = 100_000
	fee := platformFeeFromFinalPrice(goods)
	const held int64 = 1000
	wantDue := fee - held

	ar, deps, ledger, uc := newSettleWinnerFixture(clk)
	seedHeldDeposit(t, ar, deps, "auc5", "co5", held, clk.Now())

	err := uc.Execute(ctx, "auc5", "co5", goods, wantDue-1, "WINNER_FINALIZED")
	if !errors.Is(err, ErrPlatformFeeDueMismatch) {
		t.Fatalf("want ErrPlatformFeeDueMismatch got %v", err)
	}
	if len(ledger.entries) != 0 {
		t.Fatalf("expected no ledger writes, got %d", len(ledger.entries))
	}
	if deps.deposits[depKey("auc5", "co5")].Status != wallet.DepositHeld {
		t.Fatal("deposit should stay HELD on mismatch")
	}
}

func TestSettleWinnerDepositAfterInvoicePaid_idempotentSecondCall(t *testing.T) {
	ctx := context.Background()
	clk := fixedClock{t: time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)}
	const goods int64 = 100_000
	fee := platformFeeFromFinalPrice(goods)

	ar, deps, ledger, uc := newSettleWinnerFixture(clk)
	seedHeldDeposit(t, ar, deps, "auc6", "co6", fee, clk.Now())

	if err := uc.Execute(ctx, "auc6", "co6", goods, 0, "WINNER_FINALIZED"); err != nil {
		t.Fatal(err)
	}
	n := len(ledger.entries)
	if err := uc.Execute(ctx, "auc6", "co6", goods, 0, "WINNER_FINALIZED"); err != nil {
		t.Fatal(err)
	}
	if len(ledger.entries) != n {
		t.Fatalf("second call appended ledger: had %d now %d", n, len(ledger.entries))
	}
}

func TestSettleWinnerDepositAfterInvoicePaid_notHeld_noOp(t *testing.T) {
	ctx := context.Background()
	clk := fixedClock{t: time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)}
	ar, deps, ledger, uc := newSettleWinnerFixture(clk)
	acc, err := wallet.NewAccount("a-rel", "co7", wallet.CurrencyRUB)
	if err != nil {
		t.Fatal(err)
	}
	ar.accounts["co7"] = acc
	dep, err := wallet.NewAuctionDeposit("auc7", "co7", acc.ID(), 1000, wallet.CurrencyRUB, clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	dep.MarkReleased(clk.Now())
	deps.deposits[depKey("auc7", "co7")] = dep

	if err := uc.Execute(ctx, "auc7", "co7", 50_000, 9999, "WINNER_FINALIZED"); err != nil {
		t.Fatal(err)
	}
	if len(ledger.entries) != 0 {
		t.Fatalf("expected no ledger for non-HELD deposit, got %+v", ledger.entries)
	}
}

func newSettleWinnerFixture(clk fixedClock) (*memAccountRepo, *memDepositRepo, *memLedgerRepo, *SettleWinnerDepositAfterInvoicePaid) {
	ar := &memAccountRepo{accounts: map[string]*wallet.Account{}}
	deps := &memDepositRepo{deposits: map[string]*wallet.AuctionDeposit{}}
	ledger := &memLedgerRepo{}
	uc, err := NewSettleWinnerDepositAfterInvoicePaid(ar, deps, ledger, RandomHexID{}, clk, nil)
	if err != nil {
		panic(err)
	}
	return ar, deps, ledger, uc
}

func seedHeldDeposit(t *testing.T, ar *memAccountRepo, deps *memDepositRepo, auctionID, companyID string, held int64, now time.Time) {
	t.Helper()
	acc, err := wallet.NewAccount("acc:"+companyID, companyID, wallet.CurrencyRUB)
	if err != nil {
		t.Fatal(err)
	}
	if err := acc.Deposit(held + 50_000); err != nil {
		t.Fatal(err)
	}
	if err := acc.Reserve(held); err != nil {
		t.Fatal(err)
	}
	ar.accounts[companyID] = acc
	dep, err := wallet.NewAuctionDeposit(auctionID, companyID, acc.ID(), held, wallet.CurrencyRUB, now)
	if err != nil {
		t.Fatal(err)
	}
	deps.deposits[depKey(auctionID, companyID)] = dep
}
