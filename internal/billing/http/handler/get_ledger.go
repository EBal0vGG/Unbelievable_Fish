package handler

import (
	"encoding/json"
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

type GetLedgerHandler struct {
	ledger billingapp.LedgerQuery
}

func NewGetLedgerHandler(ledger billingapp.LedgerQuery) *GetLedgerHandler {
	return &GetLedgerHandler{ledger: ledger}
}

func (h *GetLedgerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	companyID, ok := identityauth.CompanyIDFromContext(r.Context())
	if !ok || companyID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	entries, err := h.ledger.ListByCompany(r.Context(), companyID, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		out = append(out, map[string]any{
			"id":             e.ID,
			"type":           string(e.EntryType),
			"amount":         e.Amount,
			"currency":       string(e.Currency),
			"reference_type": e.ReferenceType,
			"reference_id":   e.ReferenceID,
			"reason":         e.Reason,
			"created_at":     e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"entries": out})
}
