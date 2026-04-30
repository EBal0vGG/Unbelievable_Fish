package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
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

func readDealIDFromPath(path, suffix string) (string, error) {
	if !strings.HasPrefix(path, "/deals/") {
		return "", httpapi.ErrInvalidPath
	}
	rest := strings.TrimPrefix(path, "/deals/")
	parts := strings.Split(rest, "/")
	if suffix == "" {
		if len(parts) != 1 || parts[0] == "" {
			return "", httpapi.ErrInvalidPath
		}
		return parts[0], nil
	}
	tail := strings.Split(suffix, "/")
	if len(parts) != len(tail)+1 || parts[0] == "" {
		return "", httpapi.ErrInvalidPath
	}
	for i, want := range tail {
		if parts[i+1] != want {
			return "", httpapi.ErrInvalidPath
		}
	}
	return parts[0], nil
}

func readAuctionIDFromProjectionPath(path string) (string, error) {
	if !strings.HasPrefix(path, "/deal-projections/") {
		return "", httpapi.ErrInvalidPath
	}
	auctionID := strings.TrimPrefix(path, "/deal-projections/")
	if auctionID == "" || strings.Contains(auctionID, "/") {
		return "", httpapi.ErrInvalidPath
	}
	return auctionID, nil
}

func readAuctionIDFromDealPath(path string) (string, error) {
	if !strings.HasPrefix(path, "/deals/by-auction/") {
		return "", httpapi.ErrInvalidPath
	}
	auctionID := strings.TrimPrefix(path, "/deals/by-auction/")
	if auctionID == "" || strings.Contains(auctionID, "/") {
		return "", httpapi.ErrInvalidPath
	}
	return auctionID, nil
}

func readConfirmationIDsFromPath(path, action string) (string, string, error) {
	if !strings.HasPrefix(path, "/deals/") {
		return "", "", httpapi.ErrInvalidPath
	}
	rest := strings.TrimPrefix(path, "/deals/")
	parts := strings.Split(rest, "/")
	if len(parts) != 4 || parts[0] == "" || parts[1] != "confirmations" || parts[2] == "" || parts[3] != action {
		return "", "", httpapi.ErrInvalidPath
	}
	return parts[0], parts[2], nil
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
	writeJSON(w, httpErr.Status, httpapi.ErrorResponse{
		Code:          httpErr.Code,
		Message:       httpErr.Message,
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
	})
}
