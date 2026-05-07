package handler

import (
	"encoding/json"
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

type GetDepositsHandler struct {
	deposits billingapp.DepositQuery
}

func NewGetDepositsHandler(deposits billingapp.DepositQuery) *GetDepositsHandler {
	return &GetDepositsHandler{deposits: deposits}
}

func (h *GetDepositsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	companyID, ok := identityauth.CompanyIDFromContext(r.Context())
	if !ok || companyID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	items, err := h.deposits.ListByCompany(r.Context(), companyID, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	out := make([]map[string]any, 0, len(items))
	for _, d := range items {
		item := map[string]any{
			"auction_id": d.AuctionID,
			"company_id": d.CompanyID,
			"amount":     d.Amount,
			"currency":   string(d.Currency),
			"status":     string(d.Status),
			"created_at": d.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		}
		if d.ReleasedAt != nil {
			item["released_at"] = d.ReleasedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		if d.CapturedAt != nil {
			item["captured_at"] = d.CapturedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		out = append(out, item)
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"deposits": out})
}
