package handler

import (
	"context"
	"encoding/json"
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

type TestTopUpHandler struct {
	tx            TxRunner
	createAccount *billingapp.CreateAccount
	confirm       *billingapp.ConfirmTopUp
	ids           billingapp.IDGenerator
}

func NewTestTopUpHandler(tx TxRunner, create *billingapp.CreateAccount, confirm *billingapp.ConfirmTopUp, ids billingapp.IDGenerator) *TestTopUpHandler {
	if ids == nil {
		ids = billingapp.RandomHexID{}
	}
	return &TestTopUpHandler{tx: tx, createAccount: create, confirm: confirm, ids: ids}
}

type testTopUpBody struct {
	Amount int64 `json:"amount"`
}

func (h *TestTopUpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	companyID, ok := identityauth.CompanyIDFromContext(r.Context())
	if !ok || companyID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body testTopUpBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"INVALID_BODY"}`, http.StatusBadRequest)
		return
	}
	if body.Amount <= 0 {
		http.Error(w, `{"error":"INVALID_AMOUNT"}`, http.StatusBadRequest)
		return
	}
	extID := "test-topup:" + companyID + ":" + h.ids.NewID()
	if err := h.tx.WithinTx(r.Context(), func(ctx context.Context) error {
		if h.createAccount != nil {
			if err := h.createAccount.Execute(ctx, companyID); err != nil {
				return err
			}
		}
		return h.confirm.Execute(ctx, companyID, body.Amount, extID)
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
