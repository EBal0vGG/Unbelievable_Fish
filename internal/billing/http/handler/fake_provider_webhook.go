package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/payment/fake"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

// FakeProviderWebhookHandler confirms a top-up as if the payment provider called back asynchronously.
// Secured optionally via BILLING_FAKE_WEBHOOK_SECRET → header X-Billing-Fake-Webhook-Secret.
type FakeProviderWebhookHandler struct {
	tx     TxRunner
	uc     *billingapp.ConfirmTopUpByProvider
	secret string
}

func NewFakeProviderWebhookHandler(tx TxRunner, uc *billingapp.ConfirmTopUpByProvider, secret string) *FakeProviderWebhookHandler {
	return &FakeProviderWebhookHandler{tx: tx, uc: uc, secret: strings.TrimSpace(secret)}
}

type fakeProviderWebhookBody struct {
	ProviderPaymentID string `json:"provider_payment_id"`
	Amount            int64  `json:"amount"`
	Currency          string `json:"currency"`
}

func (h *FakeProviderWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	if h.secret != "" && r.Header.Get("X-Billing-Fake-Webhook-Secret") != h.secret {
		http.Error(w, `{"error":"FORBIDDEN"}`, http.StatusForbidden)
		return
	}
	var body fakeProviderWebhookBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ProviderPaymentID) == "" {
		http.Error(w, `{"error":"INVALID_BODY"}`, http.StatusBadRequest)
		return
	}
	cur := wallet.Currency(strings.ToUpper(strings.TrimSpace(body.Currency)))
	if cur == "" {
		cur = wallet.CurrencyRUB
	}
	if body.Amount <= 0 {
		http.Error(w, `{"error":"INVALID_AMOUNT"}`, http.StatusBadRequest)
		return
	}
	if err := h.tx.WithinTx(r.Context(), func(ctx context.Context) error {
		return h.uc.Execute(ctx, fake.ProviderName, body.ProviderPaymentID, body.Amount, cur)
	}); err != nil {
		if errors.Is(err, billingapp.ErrTopUpNotFound) {
			http.Error(w, `{"error":"TOP_UP_NOT_FOUND"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, billingapp.ErrTopUpAmountMismatch) || errors.Is(err, wallet.ErrInvalidTopUpStatus) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
