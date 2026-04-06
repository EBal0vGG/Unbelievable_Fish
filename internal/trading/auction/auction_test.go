package auction

import (
	"errors"
	"testing"
	"time"
)

func TestBidBeforePublishIsRejected(t *testing.T) {
	logTest(t)
	a := mustAuction(t)
	logf(t, "auction id=%s lot_id=%s state=%s starts_at=%s ends_at=%s", a.ID, a.LotID, a.State(), a.StartsAt(), a.EndsAt())

	_, err := a.PlaceBid(mustBid(t, "x", 100, time.Now()))
	logf(t, "place bid before publish error=%v", err)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBidAfterCloseIsRejected(t *testing.T) {
	logTest(t)
	a := mustAuction(t)
	logf(t, "auction id=%s lot_id=%s state=%s", a.ID, a.LotID, a.State())
	_, _ = a.Publish()
	bid := mustBid(t, "x", 100, time.Now())
	logf(t, "bid placed_at=%s amount=%d", bid.PlacedAt(), bid.Amount())
	_, _ = a.PlaceBid(bid)
	events, err := a.Close([]Bid{bid})
	logf(t, "close events=%d error=%v", len(events), err)

	_, err = a.PlaceBid(mustBid(t, "x", 100, time.Now()))
	logf(t, "place bid after close error=%v", err)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAuctionWithoutBidsIsCancelledOnClose(t *testing.T) {
	logTest(t)
	a := mustAuction(t)
	logf(t, "auction id=%s lot_id=%s state=%s", a.ID, a.LotID, a.State())
	_, _ = a.Publish()

	events, err := a.Close(nil)
	logf(t, "close events=%d error=%v state=%s", len(events), err, a.State())
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
	logTest(t)
	a := mustAuction(t)
	logf(t, "auction id=%s lot_id=%s state=%s", a.ID, a.LotID, a.State())
	_, _ = a.Publish()
	now := time.Now()
	bidA := mustBid(t, "a", 100, now)
	bidB := mustBid(t, "b", 200, now.Add(time.Second))
	logf(t, "bidA amount=%d placed_at=%s", bidA.Amount(), bidA.PlacedAt())
	logf(t, "bidB amount=%d placed_at=%s", bidB.Amount(), bidB.PlacedAt())
	_, _ = a.PlaceBid(bidA)
	_, _ = a.PlaceBid(bidB)

	events, err := a.Close([]Bid{bidA, bidB})
	logf(t, "close events=%d error=%v state=%s", len(events), err, a.State())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.State() != StateWon {
		t.Fatalf("expected state %s, got %s", StateWon, a.State())
	}
	winnerCompanyID, _, ok := a.Winner()
	logf(t, "winner_company_id=%s current_price=%d ok=%v", winnerCompanyID, a.CurrentPrice(), ok)
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
	logTest(t)
	a := mustAuction(t)
	_, _ = a.Publish()
	bid := mustBid(t, "x", 100, time.Now())
	logf(t, "bid amount=%d placed_at=%s", bid.Amount(), bid.PlacedAt())
	_, _ = a.PlaceBid(bid)
	_, _ = a.Close([]Bid{bid})

	_, err := a.Close([]Bid{bid})
	logf(t, "close second time error=%v", err)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDetermineWinner(t *testing.T) {
	logTest(t)
	t.Run("no bids", func(t *testing.T) {
		logTest(t)
		_, ok := determineWinner(nil)
		logf(t, "determineWinner ok=%v", ok)
		if ok {
			t.Fatal("expected no winner")
		}
	})

	t.Run("single bid", func(t *testing.T) {
		logTest(t)
		winner, ok := determineWinner([]Bid{mustBid(t, "a", 100, time.Now())})
		logf(t, "winner amount=%d company=%s ok=%v", winner.Amount(), winner.BidderCompanyID(), ok)
		if !ok {
			t.Fatal("expected winner")
		}
		if winner.BidderCompanyID() != "a" {
			t.Fatalf("expected winner a, got %s", winner.BidderCompanyID())
		}
	})

	t.Run("tie chooses earlier by time", func(t *testing.T) {
		logTest(t)
		now := time.Now()
		winner, ok := determineWinner([]Bid{
			mustBid(t, "a", 100, now.Add(time.Second)),
			mustBid(t, "b", 100, now),
		})
		logf(t, "winner amount=%d company=%s ok=%v", winner.Amount(), winner.BidderCompanyID(), ok)
		if !ok {
			t.Fatal("expected winner")
		}
		if winner.BidderCompanyID() != "b" {
			t.Fatalf("expected winner b, got %s", winner.BidderCompanyID())
		}
	})
}

func TestBidLowerThanCurrentPriceIsRejected(t *testing.T) {
	logTest(t)
	a := mustAuction(t)
	_, _ = a.Publish()
	now := time.Now()
	_, _ = a.PlaceBid(mustBid(t, "x", 200, now))

	_, err := a.PlaceBid(mustBid(t, "x", 100, now.Add(time.Second)))
	logf(t, "place lower bid error=%v current_price=%d", err, a.CurrentPrice())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrBidTooLow) {
		t.Fatalf("expected ErrBidTooLow, got %v", err)
	}
}

func TestBidEqualToCurrentPriceIsRejected(t *testing.T) {
	logTest(t)
	a := mustAuction(t)
	_, _ = a.Publish()
	now := time.Now()
	_, _ = a.PlaceBid(mustBid(t, "x", 200, now))

	_, err := a.PlaceBid(mustBid(t, "y", 200, now.Add(time.Second)))
	logf(t, "place equal bid error=%v current_price=%d", err, a.CurrentPrice())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrBidTooLow) {
		t.Fatalf("expected ErrBidTooLow, got %v", err)
	}
}

func TestBidExtendsAuctionNearEnd(t *testing.T) {
	logTest(t)
	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(2 * time.Minute)
	a := mustAuctionWithSchedule(t, startsAt, endsAt)
	_, _ = a.Publish()

	bidTime := endsAt.Add(-time.Minute)
	events, err := a.PlaceBid(mustBid(t, "x", 100, bidTime))
	logf(t, "bid placed_at=%s ends_at_before=%s ends_at_after=%s", bidTime, endsAt, a.EndsAt())
	logf(t, "place bid events=%d error=%v", len(events), err)
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
