package auction

import (
	"testing"
	"time"
)

func TestCannotBidAfterClose(t *testing.T) {
	a := NewAuction("1")
	_ = a.Publish()
	_ = a.PlaceBid(mustBid(t, "x", 100, time.Now()))
	_ = a.Close()

	err := a.PlaceBid(mustBid(t, "x", 100, time.Now()))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCloseWithNoBidsReturnsError(t *testing.T) {
	a := NewAuction("1")
	_ = a.Publish()

	err := a.Close()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCloseSetsWinnerAndState(t *testing.T) {
	a := NewAuction("1")
	_ = a.Publish()
	now := time.Now()
	_ = a.PlaceBid(mustBid(t, "a", 100, now))
	_ = a.PlaceBid(mustBid(t, "b", 200, now.Add(time.Second)))

	err := a.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.State() != StateWon {
		t.Fatalf("expected state %s, got %s", StateWon, a.State())
	}
	winner, ok := a.Winner()
	if !ok {
		t.Fatal("expected winner")
	}
	if winner.BidderCompanyID() != "b" {
		t.Fatalf("expected winner b, got %s", winner.BidderCompanyID())
	}
}

func TestDetermineWinner(t *testing.T) {
	t.Run("no bids", func(t *testing.T) {
		_, ok := determineWinner(nil)
		if ok {
			t.Fatal("expected no winner")
		}
	})

	t.Run("single bid", func(t *testing.T) {
		winner, ok := determineWinner([]Bid{mustBid(t, "a", 100, time.Now())})
		if !ok {
			t.Fatal("expected winner")
		}
		if winner.BidderCompanyID() != "a" {
			t.Fatalf("expected winner a, got %s", winner.BidderCompanyID())
		}
	})

	t.Run("tie chooses earlier by time", func(t *testing.T) {
		now := time.Now()
		winner, ok := determineWinner([]Bid{
			mustBid(t, "a", 100, now.Add(time.Second)),
			mustBid(t, "b", 100, now),
		})
		if !ok {
			t.Fatal("expected winner")
		}
		if winner.BidderCompanyID() != "b" {
			t.Fatalf("expected winner b, got %s", winner.BidderCompanyID())
		}
	})
}

func mustBid(t *testing.T, bidderCompanyID string, amount int64, placedAt time.Time) Bid {
	t.Helper()
	bid, err := NewBid(bidderCompanyID, amount, placedAt)
	if err != nil {
		t.Fatalf("unexpected bid error: %v", err)
	}
	return bid
}