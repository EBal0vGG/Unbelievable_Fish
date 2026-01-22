package httpapi

import (
	"errors"
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

var (
	ErrMissingCompanyID = errors.New("missing X-Company-ID header")
	ErrMissingUserID    = errors.New("missing X-User-ID header")
	ErrInvalidPath      = errors.New("invalid path")
)

type HTTPError struct {
	Status  int
	Code    string
	Message string
}

func MapError(err error) HTTPError {
	switch {
	case errors.Is(err, ErrMissingCompanyID):
		return HTTPError{http.StatusBadRequest, "MISSING_COMPANY_ID", "missing X-Company-ID header"}
	case errors.Is(err, ErrMissingUserID):
		return HTTPError{http.StatusBadRequest, "MISSING_USER_ID", "missing X-User-ID header"}
	case errors.Is(err, ErrInvalidPath):
		return HTTPError{http.StatusBadRequest, "INVALID_PATH", "invalid path"}
	case errors.Is(err, auction.ErrAuctionCannotBePublished):
		return HTTPError{http.StatusConflict, "AUCTION_NOT_PUBLISHED", "auction not in draft state"}
	case errors.Is(err, auction.ErrAuctionNotActive):
		return HTTPError{http.StatusConflict, "AUCTION_NOT_ACTIVE", "auction not active"}
	case errors.Is(err, auction.ErrCannotCloseAuction):
		return HTTPError{http.StatusConflict, "AUCTION_ALREADY_CLOSED", "auction already closed"}
	case errors.Is(err, auction.ErrInvalidStateTransition):
		return HTTPError{http.StatusConflict, "INVALID_STATE", "invalid state transition"}
	case errors.Is(err, auction.ErrCannotCancelWithBids):
		return HTTPError{http.StatusConflict, "AUCTION_HAS_BIDS", "auction has bids"}
	case errors.Is(err, auction.ErrBidderCompanyIDEmpty),
		errors.Is(err, auction.ErrBidAmountNonPositive),
		errors.Is(err, auction.ErrBidPlacedAtZero):
		return HTTPError{http.StatusBadRequest, "INVALID_BID", "invalid bid"}
	default:
		return HTTPError{http.StatusInternalServerError, "INTERNAL_ERROR", "internal error"}
	}
}

func BadRequest(code, message string) HTTPError {
	return HTTPError{http.StatusBadRequest, code, message}
}
