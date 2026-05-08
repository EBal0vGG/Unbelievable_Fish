package handler

import (
	"context"
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/go-chi/chi/v5"
)

type FakeConfirmTopUpHandler struct {
	tx TxRunner
	uc *billingapp.ConfirmTopUpByProvider
}

func NewFakeConfirmTopUpHandler(tx TxRunner, uc *billingapp.ConfirmTopUpByProvider) *FakeConfirmTopUpHandler {
	return &FakeConfirmTopUpHandler{tx: tx, uc: uc}
}

func (h *FakeConfirmTopUpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topUpID := chi.URLParam(r, "topUpID")
	if topUpID == "" {
		http.Error(w, `{"error":"MISSING_TOP_UP_ID"}`, http.StatusBadRequest)
		return
	}
	if err := h.tx.WithinTx(r.Context(), func(ctx context.Context) error {
		return h.uc.ExecuteByTopUpID(ctx, topUpID)
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
