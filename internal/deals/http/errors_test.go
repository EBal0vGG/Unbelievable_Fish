package httpapi

import (
	"net/http"
	"testing"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

func TestMapErrorAuctionDealPriceImmutable(t *testing.T) {
	got := MapError(deal.ErrAuctionDealPriceImmutable)
	if got.Status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, got.Status)
	}
	if got.Code != "DEAL_PRICE_IMMUTABLE" {
		t.Fatalf("expected DEAL_PRICE_IMMUTABLE, got %s", got.Code)
	}
}
