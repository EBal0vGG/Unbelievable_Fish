package handler

import (
	"encoding/json"
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

type ListTopUpsHandler struct {
	repo billingapp.TopUpRepository
}

func NewListTopUpsHandler(repo billingapp.TopUpRepository) *ListTopUpsHandler {
	return &ListTopUpsHandler{repo: repo}
}

func (h *ListTopUpsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	companyID, ok := identityauth.CompanyIDFromContext(r.Context())
	if !ok || companyID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	list, err := h.repo.ListByCompany(r.Context(), companyID, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, tu := range list {
		out = append(out, topUpView(tu))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"top_ups": out})
}

func topUpView(tu *wallet.TopUp) map[string]any {
	m := map[string]any{
		"id":                  tu.ID,
		"company_id":          tu.CompanyID,
		"amount":              tu.Amount,
		"currency":            string(tu.Currency),
		"status":              string(tu.Status),
		"provider":            tu.Provider,
		"provider_payment_id": tu.ProviderPaymentID,
		"confirmation_url":    tu.ConfirmationURL,
		"created_at":          tu.CreatedAt,
	}
	if tu.ConfirmedAt != nil {
		m["confirmed_at"] = *tu.ConfirmedAt
	}
	if tu.FailedAt != nil {
		m["failed_at"] = *tu.FailedAt
	}
	return m
}
