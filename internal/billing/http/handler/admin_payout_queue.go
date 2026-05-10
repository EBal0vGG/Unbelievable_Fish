package handler

import (
	"encoding/json"
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
)

// AdminListPayoutQueueHandler returns the operator payout queue (joined with invoice + company names).
type AdminListPayoutQueueHandler struct {
	lister billingapp.PayoutQueueLister
	limit  int
}

func NewAdminListPayoutQueueHandler(lister billingapp.PayoutQueueLister, limit int) *AdminListPayoutQueueHandler {
	if limit <= 0 {
		limit = 200
	}
	return &AdminListPayoutQueueHandler{lister: lister, limit: limit}
}

func (h *AdminListPayoutQueueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	rows, err := h.lister.ListPayoutQueue(r.Context(), h.limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"payouts": rows})
}
