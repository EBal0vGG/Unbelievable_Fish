package httpauth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

// ErrorResponse is the shared JSON shape for auth failures from JWT middleware.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// MapIdentityAuthError maps identity JWT / role errors to HTTP status and machine codes.
func MapIdentityAuthError(err error) (status int, code, message string) {
	switch {
	case errors.Is(err, identityauth.ErrMissingAuthorizationHeader):
		return http.StatusUnauthorized, "MISSING_AUTHORIZATION", "missing Authorization header"
	case errors.Is(err, identityauth.ErrInvalidAuthorizationHeader):
		return http.StatusUnauthorized, "INVALID_AUTHORIZATION", "invalid Authorization header"
	case errors.Is(err, identityauth.ErrInvalidToken), errors.Is(err, identityauth.ErrExpiredToken):
		return http.StatusUnauthorized, "INVALID_TOKEN", "invalid token"
	case errors.Is(err, identityauth.ErrIdentityNotFound):
		return http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized"
	case errors.Is(err, identityauth.ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN", "forbidden"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "internal error"
	}
}

// JSONErrorHandler returns an identityauth.ErrorHandler that logs and writes JSONErrorResponse.
func JSONErrorHandler(slogEventName string) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, code, msg := MapIdentityAuthError(err)
		slog.WarnContext(
			r.Context(),
			slogEventName,
			"component", "auth.middleware",
			"status", status,
			"code", code,
			"message", msg,
			"correlation_id", r.Header.Get("X-Correlation-ID"),
			"causation_id", r.Header.Get("X-Causation-ID"),
			"error", err,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(ErrorResponse{Code: code, Message: msg})
	}
}
