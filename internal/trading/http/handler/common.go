package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
)

const maxBodyBytes = 1 << 20

func readCommandMeta(r *http.Request) (app.CommandMeta, error) {
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

func readAuctionIDFromPath(path, suffix string) (app.AuctionID, error) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 3 {
		return "", httpapi.ErrInvalidPath
	}
	if parts[0] != "auctions" || parts[2] != suffix || parts[1] == "" {
		return "", httpapi.ErrInvalidPath
	}
	return app.AuctionID(parts[1]), nil
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

func requirePost(w http.ResponseWriter, r *http.Request, meta app.CommandMeta) bool {
	if r.Method == http.MethodPost {
		return true
	}
	httpErr := httpapi.MethodNotAllowed("METHOD_NOT_ALLOWED", "method not allowed")
	writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
	return false
}

func handleCommandError(w http.ResponseWriter, err error, meta app.CommandMeta) {
	httpErr := httpapi.MapError(err)
	writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
}
