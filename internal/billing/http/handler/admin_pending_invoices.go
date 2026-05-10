package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	billingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/postgres"
)

// AdminListPendingDealInvoicesHandler lists PAYMENT_PENDING deal invoices for operators.
type AdminListPendingDealInvoicesHandler struct {
	Lister *billingpg.DealInvoiceLister
	Limit  int
}

func NewAdminListPendingDealInvoicesHandler(lister *billingpg.DealInvoiceLister, limit int) *AdminListPendingDealInvoicesHandler {
	if limit <= 0 {
		limit = 200
	}
	return &AdminListPendingDealInvoicesHandler{Lister: lister, Limit: limit}
}

func (h *AdminListPendingDealInvoicesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	limit := h.Limit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	rows, err := h.Lister.ListPaymentPendingAdmin(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"invoices": rows})
}
