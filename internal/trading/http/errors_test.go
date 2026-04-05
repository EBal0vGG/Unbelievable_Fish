package httpapi

import (
	"errors"
	"net/http"
	"testing"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

func TestMapErrorNotFound(t *testing.T) {
	httpErr := MapError(app.ErrNotFound)
	if httpErr.Status != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, httpErr.Status)
	}
	if httpErr.Code != "AUCTION_NOT_FOUND" {
		t.Fatalf("expected code AUCTION_NOT_FOUND, got %s", httpErr.Code)
	}
}

func TestMapErrorContract(t *testing.T) {
	logTest(t)
	t.Run("missing company id", func(t *testing.T) {
		logTest(t)
		got := MapError(ErrMissingCompanyID)
		logf(t, "mapped=%+v", got)
		assertHTTPError(t, got, 400, "MISSING_COMPANY_ID", "missing X-Company-ID header")
	})

	t.Run("invalid state transition", func(t *testing.T) {
		logTest(t)
		got := MapError(auction.ErrInvalidStateTransition)
		logf(t, "mapped=%+v", got)
		assertHTTPError(t, got, 409, "INVALID_STATE", "invalid state transition")
	})

	t.Run("invalid bid", func(t *testing.T) {
		logTest(t)
		got := MapError(auction.ErrBidAmountNonPositive)
		logf(t, "mapped=%+v", got)
		assertHTTPError(t, got, 400, "INVALID_BID", "invalid bid")
	})

	t.Run("default", func(t *testing.T) {
		logTest(t)
		got := MapError(errors.New("boom"))
		logf(t, "mapped=%+v", got)
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
