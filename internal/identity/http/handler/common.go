package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
)

const maxBodyBytes = 1 << 20

type requestMeta struct {
	CorrelationID string
	CausationID   string
}

func readMeta(r *http.Request) requestMeta {
	return requestMeta{
		CorrelationID: r.Header.Get("X-Correlation-ID"),
		CausationID:   r.Header.Get("X-Causation-ID"),
	}
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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, err error, meta requestMeta) {
	httpErr := httpapi.MapError(err)
	logHTTPError("identity_http_error", err, httpErr.Status, httpErr.Code, httpErr.Message, meta.CorrelationID, meta.CausationID)
	writeJSON(w, httpErr.Status, httpapi.ErrorResponse{
		Code:          httpErr.Code,
		Message:       httpErr.Message,
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
	})
}

func logHTTPError(message string, err error, status int, code, responseMessage, correlationID, causationID string) {
	args := []any{
		"component", "http.handler",
		"bounded_context", "identity",
		"status", status,
		"code", code,
		"message", responseMessage,
		"correlation_id", correlationID,
		"causation_id", causationID,
		"error", err,
	}
	if status >= http.StatusInternalServerError {
		slog.Error(message, args...)
		return
	}
	slog.Warn(message, args...)
}
