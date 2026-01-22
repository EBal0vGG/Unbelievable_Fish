package httpapi

import (
	"errors"
	"testing"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

func TestMapErrorContract(t *testing.T) {
	t.Run("missing company id", func(t *testing.T) {
		got := MapError(ErrMissingCompanyID)
		assertHTTPError(t, got, 400, "MISSING_COMPANY_ID", "missing X-Company-ID header")
	})

	t.Run("invalid state transition", func(t *testing.T) {
		got := MapError(auction.ErrInvalidStateTransition)
		assertHTTPError(t, got, 409, "INVALID_STATE", "invalid state transition")
	})

	t.Run("invalid bid", func(t *testing.T) {
		got := MapError(auction.ErrBidAmountNonPositive)
		assertHTTPError(t, got, 400, "INVALID_BID", "invalid bid")
	})

	t.Run("default", func(t *testing.T) {
		got := MapError(errors.New("boom"))
		assertHTTPError(t, got, 500, "INTERNAL_ERROR", "internal error")
	})
}

func assertHTTPError(t *testing.T, got HTTPError, status int, code, message string) {
	t.Helper()
	if got.Status != status {
		t.Fatalf("expected status %d, got %d", status, got.Status)
	}
	if got.Code != code {
		t.Fatalf("expected code %s, got %s", code, got.Code)
	}
	if got.Message != message {
		t.Fatalf("expected message %s, got %s", message, got.Message)
	}
}
