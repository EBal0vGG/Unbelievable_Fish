package auction

import (
	"testing"
	"time"
)

func TestBidBeforePublishIsRejected(t *testing.T) {
	a := mustAuction(t)

	_, err := a.PlaceBid(mustBid(t, "x", 100, time.Now()))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBidAfterCloseIsRejected(t *testing.T) {
	a := mustAuction(t)
	_, _ = a.Publish()
	bid := mustBid(t, "x", 100, time.Now())
	_, _ = a.PlaceBid(bid)
	_, _ = a.Close([]Bid{bid})

	_, err := a.PlaceBid(mustBid(t, "x", 100, time.Now()))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuctionWithoutBidsIsCancelledOnClose(t *testing.T) {
	a := mustAuction(t)
	_, _ = a.Publish()

	events, err := a.Close(nil)
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
	a := mustAuction(t)
	_, _ = a.Publish()
	now := time.Now()
	bidA := mustBid(t, "a", 100, now)
	bidB := mustBid(t, "b", 200, now.Add(time.Second))
	_, _ = a.PlaceBid(bidA)
	_, _ = a.PlaceBid(bidB)

	events, err := a.Close([]Bid{bidA, bidB})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.State() != StateWon {
		t.Fatalf("expected state %s, got %s", StateWon, a.State())
	}
	winnerCompanyID, _, ok := a.Winner()
	if !ok {
		t.Fatal("expected winner")
	}
	if winnerCompanyID != "b" {
		t.Fatalf("expected winner b, got %s", winnerCompanyID)
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
	a := mustAuction(t)
	_, _ = a.Publish()
	bid := mustBid(t, "x", 100, time.Now())
	_, _ = a.PlaceBid(bid)
	_, _ = a.Close([]Bid{bid})

	_, err := a.Close([]Bid{bid})
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

func TestBidLowerThanCurrentPriceIsRejected(t *testing.T) {
	a := mustAuction(t)
	_, _ = a.Publish()
	now := time.Now()
	_, _ = a.PlaceBid(mustBid(t, "x", 200, now))

	_, err := a.PlaceBid(mustBid(t, "x", 100, now.Add(time.Second)))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBidExtendsAuctionNearEnd(t *testing.T) {
	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(2 * time.Minute)
	a := mustAuctionWithSchedule(t, startsAt, endsAt)
	_, _ = a.Publish()

	bidTime := endsAt.Add(-time.Minute)
	events, err := a.PlaceBid(mustBid(t, "x", 100, bidTime))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.EndsAt().Equal(endsAt) {
		t.Fatal("expected auction end time to be extended")
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	placed, ok := events[0].(BidPlaced)
	if !ok {
		t.Fatal("expected BidPlaced event")
	}
	if placed.NewEndAt != a.EndsAt() {
		t.Fatal("expected BidPlaced.NewEndAt to match auction end time")
	}
}

func mustAuction(t *testing.T) *Auction {
	t.Helper()
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(2 * time.Hour)
	return mustAuctionWithSchedule(t, startsAt, endsAt)
}

func mustAuctionWithSchedule(t *testing.T, startsAt, endsAt time.Time) *Auction {
	t.Helper()
	a, err := NewAuction("1", "lot-1", startsAt, endsAt)
	if err != nil {
		t.Fatalf("unexpected auction error: %v", err)
	}
	return a
}

func mustBid(t *testing.T, bidderCompanyID string, amount int64, placedAt time.Time) Bid {
	t.Helper()
	bid, err := NewBid(bidderCompanyID, amount, placedAt)
	if err != nil {
		t.Fatalf("unexpected bid error: %v", err)
	}
	return bid
}
