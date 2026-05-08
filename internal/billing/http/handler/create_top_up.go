package handler

import (
	"context"
	"encoding/json"
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

type CreateTopUpHandler struct {
	tx TxRunner
	uc *billingapp.CreateTopUp
}

func NewCreateTopUpHandler(tx TxRunner, uc *billingapp.CreateTopUp) *CreateTopUpHandler {
	return &CreateTopUpHandler{tx: tx, uc: uc}
}

type createTopUpJSON struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

func (h *CreateTopUpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	companyID, ok := identityauth.CompanyIDFromContext(r.Context())
	if !ok || companyID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body createTopUpJSON
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"INVALID_BODY"}`, http.StatusBadRequest)
		return
	}
	if body.Amount <= 0 {
		http.Error(w, `{"error":"INVALID_AMOUNT"}`, http.StatusBadRequest)
		return
	}
	cur := wallet.Currency(body.Currency)
	if body.Currency == "" {
		cur = wallet.CurrencyRUB
	}
	var tu *wallet.TopUp
	if err := h.tx.WithinTx(r.Context(), func(ctx context.Context) error {
		var err error
		tu, err = h.uc.Execute(ctx, companyID, body.Amount, cur)
		return err
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"top_up_id":        tu.ID,
		"status":           string(tu.Status),
		"amount":           tu.Amount,
		"currency":         string(tu.Currency),
		"confirmation_url": tu.ConfirmationURL,
	})
}
