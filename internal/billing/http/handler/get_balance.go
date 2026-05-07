package handler

import (
	"context"
	"encoding/json"
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

type GetBalanceHandler struct {
	tx            TxRunner
	accounts      billingapp.AccountRepository
	createAccount *billingapp.CreateAccount
}

func NewGetBalanceHandler(tx TxRunner, accounts billingapp.AccountRepository, create *billingapp.CreateAccount) *GetBalanceHandler {
	return &GetBalanceHandler{tx: tx, accounts: accounts, createAccount: create}
}

func (h *GetBalanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	companyID, ok := identityauth.CompanyIDFromContext(r.Context())
	if !ok || companyID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.createAccount != nil {
		if err := h.tx.WithinTx(r.Context(), func(ctx context.Context) error {
			return h.createAccount.Execute(ctx, companyID)
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	acc, err := h.accounts.LoadByCompany(r.Context(), companyID)
	if err != nil {
		if err == billingapp.ErrAccountNotFound {
			http.Error(w, `{"error":"ACCOUNT_NOT_FOUND"}`, http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"company_id": acc.CompanyID(),
		"currency":   string(acc.Currency()),
		"available":  acc.Available(),
		"held":       acc.Held(),
		"total":      acc.Total(),
	})
}
