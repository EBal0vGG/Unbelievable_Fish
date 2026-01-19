package auction

import (
	"testing"
)
	
func TestCannotBidAfterClose(t *testing.T) {
	a := NewAuction("1")
	_ = a.Publish()
	_ = a.Close()

	err := a.PlaceBid(Bid{Amount: 100})
	if err == nil {
		t.Fatal("expected error")
	}
}