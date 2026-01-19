package auction

import (
	"testing"
	"time"
)

func TestNewBidValidation(t *testing.T) {
	t.Run("empty bidder company id", func(t *testing.T) {
		_, err := NewBid("", 100, time.Now())
		if err != ErrBidderCompanyIDEmpty {
			t.Fatalf("expected %v, got %v", ErrBidderCompanyIDEmpty, err)
		}
	})

	t.Run("non-positive amount", func(t *testing.T) {
		_, err := NewBid("a", 0, time.Now())
		if err != ErrBidAmountNonPositive {
			t.Fatalf("expected %v, got %v", ErrBidAmountNonPositive, err)
		}
	})

	t.Run("zero placed at", func(t *testing.T) {
		_, err := NewBid("a", 100, time.Time{})
		if err != ErrBidPlacedAtZero {
			t.Fatalf("expected %v, got %v", ErrBidPlacedAtZero, err)
		}
	})
}
