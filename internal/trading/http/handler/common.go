package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
)

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
	if !strings.HasPrefix(path, "/auctions/") {
		return "", httpapi.ErrInvalidPath
	}
	rest := strings.TrimPrefix(path, "/auctions/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != suffix || parts[0] == "" {
		return "", httpapi.ErrInvalidPath
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
