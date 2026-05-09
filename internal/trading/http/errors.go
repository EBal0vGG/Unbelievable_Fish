package httpapi

import (
	"errors"
	"net/http"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
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
	case errors.Is(err, identityauth.ErrMissingAuthorizationHeader):
		return HTTPError{http.StatusUnauthorized, "MISSING_AUTHORIZATION", "missing Authorization header"}
	case errors.Is(err, identityauth.ErrInvalidAuthorizationHeader):
		return HTTPError{http.StatusUnauthorized, "INVALID_AUTHORIZATION", "invalid Authorization header"}
	case errors.Is(err, identityauth.ErrInvalidToken), errors.Is(err, identityauth.ErrExpiredToken):
		return HTTPError{http.StatusUnauthorized, "INVALID_TOKEN", "invalid token"}
	case errors.Is(err, identityauth.ErrForbidden):
		return HTTPError{http.StatusForbidden, "FORBIDDEN", "forbidden"}
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
	case errors.Is(err, auction.ErrAuctionNotStarted):
		return HTTPError{http.StatusConflict, "AUCTION_NOT_ACTIVE", "auction not started"}
	case errors.Is(err, auction.ErrAuctionAlreadyEnded):
		return HTTPError{http.StatusConflict, "AUCTION_NOT_ACTIVE", "auction already ended"}
	case errors.Is(err, auction.ErrCannotCloseAuction):
		return HTTPError{http.StatusConflict, "AUCTION_ALREADY_CLOSED", "auction already closed"}
	case errors.Is(err, auction.ErrInvalidStateTransition):
		return HTTPError{http.StatusConflict, "INVALID_STATE", "invalid state transition"}
	case errors.Is(err, auction.ErrCannotCancelWithBids):
		return HTTPError{http.StatusConflict, "AUCTION_HAS_BIDS", "auction has bids"}
	case errors.Is(err, auction.ErrAuctionIDEmpty),
		errors.Is(err, auction.ErrLotIDEmpty):
		return HTTPError{http.StatusBadRequest, "INVALID_BODY", "invalid request body"}
	case errors.Is(err, auction.ErrBidderCompanyIDEmpty),
		errors.Is(err, auction.ErrBidAmountNonPositive),
		errors.Is(err, auction.ErrBidPlacedAtZero),
		errors.Is(err, auction.ErrBidTooLow):
		return HTTPError{http.StatusBadRequest, "INVALID_BID", "invalid bid"}
	case errors.Is(err, auction.ErrBidStepTooSmall):
		return HTTPError{http.StatusBadRequest, "BID_TOO_SMALL", "bid is below minimum allowed"}
	case errors.Is(err, auction.ErrInvalidSchedule):
		return HTTPError{http.StatusBadRequest, "INVALID_SCHEDULE", "invalid auction schedule"}
	case errors.Is(err, auction.ErrInvalidStartPrice),
		errors.Is(err, auction.ErrInvalidMinBidStep):
		return HTTPError{http.StatusBadRequest, "INVALID_BODY", "invalid request body"}
	case errors.Is(err, app.ErrNotFound):
		return HTTPError{http.StatusNotFound, "AUCTION_NOT_FOUND", "auction not found"}
	case errors.Is(err, app.ErrInsufficientFundsForDeposit):
		return HTTPError{http.StatusConflict, "INSUFFICIENT_FUNDS_FOR_DEPOSIT", "insufficient funds for auction deposit; top up your balance"}
	default:
		return HTTPError{http.StatusInternalServerError, "INTERNAL_ERROR", "internal error"}
	}
}

func BadRequest(code, message string) HTTPError {
	return HTTPError{http.StatusBadRequest, code, message}
}
