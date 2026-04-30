package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	"github.com/go-chi/chi/v5"
)

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

func readDealIDFromRequest(r *http.Request) (string, error) {
	dealID := chi.URLParam(r, "dealID")
	if dealID != "" {
		return dealID, nil
	}
	return "", httpapi.ErrInvalidPath
}

func readAuctionIDFromRequest(r *http.Request) (string, error) {
	auctionID := chi.URLParam(r, "auctionID")
	if auctionID != "" {
		return auctionID, nil
	}
	return "", httpapi.ErrInvalidPath
}

func readConfirmationIDsFromRequest(r *http.Request) (string, string, error) {
	dealID := chi.URLParam(r, "dealID")
	confirmationID := chi.URLParam(r, "confirmationID")
	if dealID == "" || confirmationID == "" {
		return "", "", httpapi.ErrInvalidPath
	}
	return dealID, confirmationID, nil
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeAccepted(w http.ResponseWriter) {
	w.WriteHeader(http.StatusAccepted)
}

func writeCreatedJSON(w http.ResponseWriter, payload any) {
	writeJSON(w, http.StatusCreated, payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, err error, meta app.CommandMeta) {
	httpErr := httpapi.MapError(err)
	logHTTPError("deals_http_error", err, httpErr.Status, httpErr.Code, httpErr.Message, meta)
	writeJSON(w, httpErr.Status, httpapi.ErrorResponse{
		Code:          httpErr.Code,
		Message:       httpErr.Message,
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
	})
}

func logHTTPError(message string, err error, status int, code, responseMessage string, meta app.CommandMeta) {
	args := []any{
		"component", "http.handler",
		"bounded_context", "deals",
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
