package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
)

var (
	errMissingCompanyID = errors.New("missing X-Company-ID header")
	errMissingUserID    = errors.New("missing X-User-ID header")
	errInvalidPath      = errors.New("invalid path")
)

func readCommandMeta(r *http.Request) (app.CommandMeta, error) {
	companyID := r.Header.Get("X-Company-ID")
	if companyID == "" {
		return app.CommandMeta{}, errMissingCompanyID
	}
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		return app.CommandMeta{}, errMissingUserID
	}
	return app.CommandMeta{
		CompanyID:     companyID,
		UserID:        userID,
		CorrelationID: r.Header.Get("X-Correlation-ID"),
		CausationID:   r.Header.Get("X-Causation-ID"),
	}, nil
}

func readAuctionIDFromPath(path, suffix string) (app.AuctionID, error) {
	if !strings.HasPrefix(path, "/auctions/") {
		return "", errInvalidPath
	}
	rest := strings.TrimPrefix(path, "/auctions/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != suffix || parts[0] == "" {
		return "", errInvalidPath
	}
	return app.AuctionID(parts[0]), nil
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeError(w http.ResponseWriter, status int, code, message string, meta app.CommandMeta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(httpapi.ErrorResponse{
		Code:          code,
		Message:       message,
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
	})
}

func mapError(err error) (int, string, string) {
	switch {
	case errors.Is(err, errMissingCompanyID):
		return http.StatusBadRequest, "MISSING_COMPANY_ID", "missing X-Company-ID header"
	case errors.Is(err, errMissingUserID):
		return http.StatusBadRequest, "MISSING_USER_ID", "missing X-User-ID header"
	case errors.Is(err, errInvalidPath):
		return http.StatusBadRequest, "INVALID_PATH", "invalid path"
	case errors.Is(err, auction.ErrAuctionCannotBePublished):
		return http.StatusConflict, "AUCTION_NOT_PUBLISHED", "auction not in draft state"
	case errors.Is(err, auction.ErrAuctionNotActive):
		return http.StatusConflict, "AUCTION_NOT_ACTIVE", "auction not active"
	case errors.Is(err, auction.ErrCannotCloseAuction):
		return http.StatusConflict, "AUCTION_ALREADY_CLOSED", "auction already closed"
	case errors.Is(err, auction.ErrInvalidStateTransition):
		return http.StatusConflict, "INVALID_STATE_TRANSITION", "invalid state transition"
	case errors.Is(err, auction.ErrCannotCancelWithBids):
		return http.StatusConflict, "AUCTION_HAS_BIDS", "auction has bids"
	case errors.Is(err, auction.ErrBidderCompanyIDEmpty),
		errors.Is(err, auction.ErrBidAmountNonPositive),
		errors.Is(err, auction.ErrBidPlacedAtZero):
		return http.StatusBadRequest, "INVALID_BID", "invalid bid"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal error"
	}
}
