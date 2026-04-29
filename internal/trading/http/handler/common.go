package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
	"github.com/go-chi/chi/v5"
)

const maxBodyBytes = 1 << 20

func readCommandMeta(r *http.Request) (app.CommandMeta, error) {
	if identity, ok := identityauth.IdentityFromContext(r.Context()); ok {
		return app.CommandMeta{
			CompanyID:     identity.CompanyID,
			UserID:        identity.UserID,
			CorrelationID: r.Header.Get("X-Correlation-ID"),
			CausationID:   r.Header.Get("X-Causation-ID"),
		}, nil
	}
	companyID := r.Header.Get("X-Company-ID")
	if companyID == "" {
		return app.CommandMeta{}, httpapi.ErrMissingCompanyID
	}
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		return app.CommandMeta{}, httpapi.ErrMissingUserID
	}
	return app.CommandMeta{
		CompanyID:     companyID,
		UserID:        userID,
		CorrelationID: r.Header.Get("X-Correlation-ID"),
		CausationID:   r.Header.Get("X-Causation-ID"),
	}, nil
}

func readAuctionIDFromRequest(r *http.Request) (app.AuctionID, error) {
	auctionID := chi.URLParam(r, "auctionID")
	if auctionID != "" {
		return app.AuctionID(auctionID), nil
	}
	return "", httpapi.ErrInvalidPath
}

func readLotIDFromRequest(r *http.Request) (string, error) {
	lotID := chi.URLParam(r, "lotID")
	if lotID != "" {
		return lotID, nil
	}
	return "", httpapi.ErrInvalidPath
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values")
	}
	return nil
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

func writeAccepted(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}

func writeAcceptedJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(payload)
}

func handleCommandError(w http.ResponseWriter, err error, meta app.CommandMeta) {
	httpErr := httpapi.MapError(err)
	logHTTPError("trading_http_error", err, httpErr.Status, httpErr.Code, httpErr.Message, meta)
	writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
}

func logHTTPError(message string, err error, status int, code, responseMessage string, meta app.CommandMeta) {
	args := []any{
		"component", "http.handler",
		"bounded_context", "trading",
		"status", status,
		"code", code,
		"message", responseMessage,
		"company_id", meta.CompanyID,
		"user_id", meta.UserID,
		"correlation_id", meta.CorrelationID,
		"causation_id", meta.CausationID,
		"error", err,
	}
	if status >= http.StatusInternalServerError {
		slog.Error(message, args...)
		return
	}
	slog.Warn(message, args...)
}
