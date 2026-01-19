package auction

import (
	"testing"
	"time"
)

func TestBidBeforePublishIsRejected(t *testing.T) {
	a := NewAuction("1")

	_, err := a.PlaceBid(mustBid(t, "x", 100, time.Now()))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBidAfterCloseIsRejected(t *testing.T) {
	a := NewAuction("1")
	_, _ = a.Publish()
	_, _ = a.PlaceBid(mustBid(t, "x", 100, time.Now()))
	_, _ = a.Close()

	_, err := a.PlaceBid(mustBid(t, "x", 100, time.Now()))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuctionWithoutBidsIsCancelledOnClose(t *testing.T) {
	a := NewAuction("1")
	_, _ = a.Publish()

	events, err := a.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.State() != StateCancelled {
		t.Fatalf("expected state %s, got %s", StateCancelled, a.State())
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if _, ok := events[0].(AuctionCancelled); !ok {
		t.Fatal("expected AuctionCancelled event")
	}
}

func TestAuctionWithBidsIsWonOnClose(t *testing.T) {
	a := NewAuction("1")
	_, _ = a.Publish()
	now := time.Now()
	_, _ = a.PlaceBid(mustBid(t, "a", 100, now))
	_, _ = a.PlaceBid(mustBid(t, "b", 200, now.Add(time.Second)))

	events, err := a.Close()
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
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if _, ok := events[0].(AuctionClosed); !ok {
		t.Fatal("expected AuctionClosed event")
	}
	if _, ok := events[1].(AuctionWon); !ok {
		t.Fatal("expected AuctionWon event")
	}
}

func TestClosingAuctionTwiceIsRejected(t *testing.T) {
	a := NewAuction("1")
	_, _ = a.Publish()
	_, _ = a.PlaceBid(mustBid(t, "x", 100, time.Now()))
	_, _ = a.Close()

	_, err := a.Close()
	if err == nil {
		t.Fatal("expected error")
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
