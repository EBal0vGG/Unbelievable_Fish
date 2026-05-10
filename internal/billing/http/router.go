package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	GetBalance             http.Handler
	TestTopUp              http.Handler
	GetLedger              http.Handler
	GetDeposits            http.Handler
	CreateTopUp            http.Handler
	ListTopUps             http.Handler
	FakeConfirmTopUp       http.Handler
	FakeProviderWebhook    http.Handler
	GetDealInvoice         http.Handler
	GetDealInvoiceByDeal   http.Handler
	ListMyDealInvoices     http.Handler
	FakeConfirmDealInvoice http.Handler
	ListMySellerPayouts    http.Handler
	GetSellerPayout        http.Handler
	AdminConfirmDealInvoice http.Handler
	AdminExpireDealInvoice  http.Handler
	AdminListPendingDealInvoices http.Handler
	AdminListPayoutQueue    http.Handler
	AdminMarkPayoutReady    http.Handler
	AdminMarkPayoutPaid     http.Handler
	AdminMarkPayoutFailed   http.Handler
}

func NewRouter(h Handlers, middlewares ...func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares...)
	r.Route("/accounts/me", func(r chi.Router) {
		r.Method(http.MethodGet, "/", h.GetBalance)
		r.Method(http.MethodPost, "/top-up/test", h.TestTopUp)
		r.Method(http.MethodGet, "/ledger", h.GetLedger)
		r.Method(http.MethodGet, "/deposits", h.GetDeposits)
	})
	r.Route("/webhooks", func(r chi.Router) {
		r.Method(http.MethodPost, "/fake-provider", h.FakeProviderWebhook)
	})
	r.Route("/top-ups", func(r chi.Router) {
		r.Method(http.MethodPost, "/", h.CreateTopUp)
		r.Method(http.MethodGet, "/", h.ListTopUps)
		r.Method(http.MethodPost, "/{topUpID}/fake-confirm", h.FakeConfirmTopUp)
	})
	r.Route("/invoices", func(r chi.Router) {
		r.Method(http.MethodGet, "/me", h.ListMyDealInvoices)
		r.Method(http.MethodGet, "/by-deal/{dealID}", h.GetDealInvoiceByDeal)
		r.Method(http.MethodGet, "/{invoiceID}", h.GetDealInvoice)
		r.Method(http.MethodPost, "/{invoiceID}/fake-confirm", h.FakeConfirmDealInvoice)
	})
	r.Route("/payouts", func(r chi.Router) {
		r.Method(http.MethodGet, "/me", h.ListMySellerPayouts)
		r.Method(http.MethodGet, "/{payoutID}", h.GetSellerPayout)
	})
	// /admin/* handlers are swapped in cmd/billing (RequireRole(admin) vs 404). When adding routes here,
	// wire them the same way so auth is never only "forgotten" in the router layer.
	r.Route("/admin", func(r chi.Router) {
		r.Route("/invoices", func(r chi.Router) {
			r.Method(http.MethodGet, "/pending", h.AdminListPendingDealInvoices)
			r.Method(http.MethodPost, "/{invoiceID}/confirm", h.AdminConfirmDealInvoice)
			r.Method(http.MethodPost, "/{invoiceID}/expire", h.AdminExpireDealInvoice)
		})
		r.Route("/payouts", func(r chi.Router) {
			r.Method(http.MethodGet, "/", h.AdminListPayoutQueue)
			r.Method(http.MethodPost, "/{payoutID}/ready", h.AdminMarkPayoutReady)
			r.Method(http.MethodPost, "/{payoutID}/paid", h.AdminMarkPayoutPaid)
			r.Method(http.MethodPost, "/{payoutID}/failed", h.AdminMarkPayoutFailed)
		})
	})
	return r
}
