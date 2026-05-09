package deal

import (
	"errors"
	"testing"
	"time"
)

func TestWinnerSelection_MarkCurrentConfirmed(t *testing.T) {
	snap := ProductSnapshot{ProductID: "p", Name: "n"}
	s := NewWinnerSelection("auc-1", []string{"buyer-1", "buyer-2"}, 100, time.Now(), "sup-1", snap)
	s.DealID = "deal-1"

	if err := s.MarkCurrentConfirmed("deal-1"); err != nil {
		t.Fatalf("MarkCurrentConfirmed: %v", err)
	}
	if s.Status != WinnerSelectionConfirmedPendingPayment {
		t.Fatalf("status: got %q want %q", s.Status, WinnerSelectionConfirmedPendingPayment)
	}
}

func TestWinnerSelection_MarkCurrentConfirmed_staleDealID(t *testing.T) {
	snap := ProductSnapshot{ProductID: "p", Name: "n"}
	s := NewWinnerSelection("auc-1", []string{"buyer-1"}, 100, time.Now(), "sup-1", snap)
	s.DealID = "deal-2"

	if err := s.MarkCurrentConfirmed("deal-1"); err == nil || !errors.Is(err, ErrStaleWinnerSelection) {
		t.Fatalf("want ErrStaleWinnerSelection, got %v", err)
	}
}

func TestWinnerSelection_MarkCurrentConfirmed_assignsDealIDWhenEmpty(t *testing.T) {
	snap := ProductSnapshot{ProductID: "p", Name: "n"}
	s := NewWinnerSelection("auc-1", []string{"buyer-1"}, 100, time.Now(), "sup-1", snap)

	if err := s.MarkCurrentConfirmed("deal-1"); err != nil {
		t.Fatalf("MarkCurrentConfirmed: %v", err)
	}
	if s.DealID != "deal-1" {
		t.Fatalf("DealID: got %q", s.DealID)
	}
	if s.Status != WinnerSelectionConfirmedPendingPayment {
		t.Fatalf("status: %s", s.Status)
	}
}

func TestWinnerSelection_Advance_exhaustedKeepsIndexInBounds(t *testing.T) {
	snap := ProductSnapshot{ProductID: "p", Name: "n"}
	s := NewWinnerSelection("auc-1", []string{"buyer-1", "buyer-2"}, 100, time.Now(), "sup-1", snap)
	s.CurrentIndex = 1
	if s.Advance() {
		t.Fatal("expected Advance false at last candidate")
	}
	if s.Status != WinnerSelectionExhausted {
		t.Fatalf("status: %s", s.Status)
	}
	if s.CurrentIndex != 1 {
		t.Fatalf("CurrentIndex: got %d want 1 (must stay within Candidates)", s.CurrentIndex)
	}
}

func TestWinnerSelection_Advance_singleCandidateExhaustsWithoutPastEnd(t *testing.T) {
	snap := ProductSnapshot{ProductID: "p", Name: "n"}
	s := NewWinnerSelection("auc-1", []string{"buyer-1"}, 100, time.Now(), "sup-1", snap)
	if s.Advance() {
		t.Fatal("expected Advance false")
	}
	if s.Status != WinnerSelectionExhausted {
		t.Fatalf("status: %s", s.Status)
	}
	if s.CurrentIndex != 0 {
		t.Fatalf("CurrentIndex: got %d want 0", s.CurrentIndex)
	}
}

func TestWinnerSelection_Advance_notFromConfirmedPendingPayment(t *testing.T) {
	snap := ProductSnapshot{ProductID: "p", Name: "n"}
	s := NewWinnerSelection("auc-1", []string{"buyer-1", "buyer-2"}, 100, time.Now(), "sup-1", snap)
	s.DealID = "deal-1"
	s.Status = WinnerSelectionConfirmedPendingPayment
	s.CurrentIndex = 0

	if s.Advance() {
		t.Fatal("Advance should return false when not active")
	}
	if s.CurrentIndex != 0 {
		t.Fatalf("index changed: %d", s.CurrentIndex)
	}
}

func TestWinnerSelection_CurrentCandidate_confirmedPendingPayment(t *testing.T) {
	snap := ProductSnapshot{ProductID: "p", Name: "n"}
	s := NewWinnerSelection("auc-1", []string{"buyer-1"}, 100, time.Now(), "sup-1", snap)
	s.Status = WinnerSelectionConfirmedPendingPayment
	c, ok := s.CurrentCandidate()
	if !ok || c != "buyer-1" {
		t.Fatalf("CurrentCandidate: ok=%v c=%q", ok, c)
	}
}
