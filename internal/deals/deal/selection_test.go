package deal

import (
	"errors"
	"testing"
	"time"
)

func TestWinnerSelection_MarkFinalized_happyPath(t *testing.T) {
	sel := NewWinnerSelection("auc-1", []string{"buyer-1"}, 200, time.Now().UTC(), "sup-1", ProductSnapshot{Name: "x"})
	sel.DealID = "deal-1"
	sel.Status = WinnerSelectionConfirmedPendingPayment
	if err := sel.MarkFinalized("deal-1"); err != nil {
		t.Fatal(err)
	}
	if sel.Status != WinnerSelectionStatusFinalized {
		t.Fatalf("status: want finalized got %s", sel.Status)
	}
	if sel.DealID != "deal-1" {
		t.Fatalf("DealID should remain: %q", sel.DealID)
	}
}

func TestWinnerSelection_MarkFinalized_idempotentSameDeal(t *testing.T) {
	sel := NewWinnerSelection("auc-1", []string{"buyer-1"}, 200, time.Now().UTC(), "sup-1", ProductSnapshot{Name: "x"})
	sel.DealID = "deal-1"
	sel.Status = WinnerSelectionConfirmedPendingPayment
	if err := sel.MarkFinalized("deal-1"); err != nil {
		t.Fatal(err)
	}
	if err := sel.MarkFinalized("deal-1"); err != nil {
		t.Fatal(err)
	}
}

func TestWinnerSelection_MarkFinalized_wrongDealID(t *testing.T) {
	sel := NewWinnerSelection("auc-1", []string{"buyer-1"}, 200, time.Now().UTC(), "sup-1", ProductSnapshot{Name: "x"})
	sel.DealID = "deal-1"
	sel.Status = WinnerSelectionConfirmedPendingPayment
	err := sel.MarkFinalized("other-deal")
	if !errors.Is(err, ErrWinnerSelectionDealMismatch) {
		t.Fatalf("want ErrWinnerSelectionDealMismatch got %v", err)
	}
}

func TestWinnerSelection_MarkFinalized_wrongStatus(t *testing.T) {
	sel := NewWinnerSelection("auc-1", []string{"buyer-1"}, 200, time.Now().UTC(), "sup-1", ProductSnapshot{Name: "x"})
	sel.DealID = "deal-1"
	sel.Status = WinnerSelectionActive
	err := sel.MarkFinalized("deal-1")
	if !errors.Is(err, ErrWinnerSelectionNotAwaitingPayment) {
		t.Fatalf("want ErrWinnerSelectionNotAwaitingPayment got %v", err)
	}
}

func TestWinnerSelection_Advance_clearsDealIDWhenMovingToNextCandidate(t *testing.T) {
	sel := NewWinnerSelection("auc-1", []string{"buyer-1", "buyer-2"}, 200, time.Now().UTC(), "sup-1", ProductSnapshot{Name: "x"})
	sel.DealID = "deal-old"
	if !sel.Advance() {
		t.Fatal("expected advance to next candidate")
	}
	if sel.DealID != "" {
		t.Fatalf("DealID should clear until new deal is bound, got %q", sel.DealID)
	}
	if sel.CurrentIndex != 1 {
		t.Fatalf("CurrentIndex: want 1 got %d", sel.CurrentIndex)
	}
}

func TestWinnerSelection_Advance_exhaustedClearsDealID(t *testing.T) {
	sel := NewWinnerSelection("auc-1", []string{"buyer-1"}, 200, time.Now().UTC(), "sup-1", ProductSnapshot{Name: "x"})
	sel.DealID = "deal-1"
	if sel.Advance() {
		t.Fatal("expected no further candidate")
	}
	if sel.Status != WinnerSelectionExhausted {
		t.Fatalf("status: want exhausted got %s", sel.Status)
	}
	if sel.DealID != "" {
		t.Fatalf("DealID should clear when exhausted, got %q", sel.DealID)
	}
}

func TestWinnerSelection_MarkExhausted_clearsDealID(t *testing.T) {
	sel := NewWinnerSelection("auc-1", []string{"buyer-1"}, 200, time.Now().UTC(), "sup-1", ProductSnapshot{Name: "x"})
	sel.DealID = "deal-1"
	sel.MarkExhausted()
	if sel.DealID != "" {
		t.Fatalf("DealID should clear, got %q", sel.DealID)
	}
	if sel.Status != WinnerSelectionExhausted {
		t.Fatalf("status: want exhausted got %s", sel.Status)
	}
}

func TestWinnerSelection_ReopenAfterPaymentTimeout(t *testing.T) {
	sel := NewWinnerSelection("auc-1", []string{"buyer-1", "buyer-2"}, 200, time.Now().UTC(), "sup-1", ProductSnapshot{Name: "x"})
	sel.DealID = "deal-1"
	sel.Status = WinnerSelectionConfirmedPendingPayment
	if err := sel.ReopenAfterPaymentTimeout("deal-1"); err != nil {
		t.Fatal(err)
	}
	if sel.Status != WinnerSelectionActive || sel.DealID != "" {
		t.Fatalf("want active and empty DealID, got status=%s dealID=%q", sel.Status, sel.DealID)
	}
	if sel.CurrentIndex != 0 {
		t.Fatalf("index should stay 0 before Advance, got %d", sel.CurrentIndex)
	}
}

func TestWinnerSelection_ReopenAfterPaymentTimeout_wrongDeal(t *testing.T) {
	sel := NewWinnerSelection("auc-1", []string{"buyer-1"}, 200, time.Now().UTC(), "sup-1", ProductSnapshot{Name: "x"})
	sel.DealID = "deal-1"
	sel.Status = WinnerSelectionConfirmedPendingPayment
	err := sel.ReopenAfterPaymentTimeout("other")
	if !errors.Is(err, ErrWinnerSelectionDealMismatch) {
		t.Fatalf("want ErrWinnerSelectionDealMismatch got %v", err)
	}
}

func TestWinnerSelection_MarkFinalized_finalizedOtherDeal(t *testing.T) {
	sel := NewWinnerSelection("auc-1", []string{"buyer-1"}, 200, time.Now().UTC(), "sup-1", ProductSnapshot{Name: "x"})
	sel.DealID = "deal-1"
	sel.Status = WinnerSelectionConfirmedPendingPayment
	if err := sel.MarkFinalized("deal-1"); err != nil {
		t.Fatal(err)
	}
	err := sel.MarkFinalized("deal-2")
	if !errors.Is(err, ErrWinnerSelectionDealMismatch) {
		t.Fatalf("want ErrWinnerSelectionDealMismatch got %v", err)
	}
}
