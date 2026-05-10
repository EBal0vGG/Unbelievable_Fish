package fake

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
)

const ProviderName = "fake"

// Provider is a dev-only payment provider that builds deterministic payment ids and return URLs.
// Optional AutoWebhook simulates an asynchronous provider callback to the billing webhook endpoint.
type Provider struct {
	AutoWebhook    bool
	WebhookDelay   time.Duration
	WebhookBaseURL string // e.g. http://127.0.0.1:8085/billing (no trailing slash)
	WebhookSecret  string // sent as X-Billing-Fake-Webhook-Secret when non-empty
	HTTPClient     *http.Client
}

func (p Provider) CreateTopUp(ctx context.Context, req billingapp.CreateTopUpRequest) (billingapp.CreateTopUpResponse, error) {
	_ = ctx
	pid := "fake-pay-" + req.TopUpID
	confirmationURL := "/billing/top-ups/" + req.TopUpID + "/fake-confirm"
	resp := billingapp.CreateTopUpResponse{
		ProviderPaymentID: pid,
		ConfirmationURL:   confirmationURL,
	}
	if p.AutoWebhook && strings.TrimSpace(p.WebhookBaseURL) != "" {
		delay := p.WebhookDelay
		if delay <= 0 {
			delay = 2 * time.Second
		}
		client := p.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		base := strings.TrimRight(strings.TrimSpace(p.WebhookBaseURL), "/")
		secret := p.WebhookSecret
		amount := req.Amount
		cur := string(req.Currency)
		if cur == "" {
			cur = "RUB"
		}
		go p.postTopUpWebhook(delay, client, base, secret, pid, amount, cur)
	}
	return resp, nil
}

func (p Provider) postTopUpWebhook(delay time.Duration, client *http.Client, base, secret, providerPaymentID string, amount int64, currency string) {
	time.Sleep(delay)
	payload := map[string]any{
		"provider_payment_id": providerPaymentID,
		"amount":              amount,
		"currency":            currency,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	reqOut, err := http.NewRequest(http.MethodPost, base+"/webhooks/fake-provider", bytes.NewReader(body))
	if err != nil {
		return
	}
	reqOut.Header.Set("Content-Type", "application/json")
	if secret != "" {
		reqOut.Header.Set("X-Billing-Fake-Webhook-Secret", secret)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	reqOut = reqOut.WithContext(ctx)
	_, _ = client.Do(reqOut)
}

func (Provider) CreateDealInvoice(ctx context.Context, req billingapp.CreateDealInvoiceRequest) (billingapp.CreateDealInvoiceResponse, error) {
	_ = ctx
	pid := "fake-inv-" + req.InvoiceID
	rel := "/billing/invoices/" + req.InvoiceID + "/fake-confirm"
	paymentURL := rel
	if b := strings.TrimRight(req.ReturnURL, "/"); b != "" {
		paymentURL = b + rel
	}
	return billingapp.CreateDealInvoiceResponse{
		ProviderInvoiceID: pid,
		PaymentURL:        paymentURL,
	}, nil
}
