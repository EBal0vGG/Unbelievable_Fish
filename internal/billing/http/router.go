package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	GetBalance            http.Handler
	TestTopUp             http.Handler
	GetLedger             http.Handler
	GetDeposits           http.Handler
	CreateTopUp           http.Handler
	ListTopUps            http.Handler
	FakeConfirmTopUp      http.Handler
	GetDealInvoice        http.Handler
	GetDealInvoiceByDeal  http.Handler
	ListMyDealInvoices    http.Handler
	FakeConfirmDealInvoice http.Handler
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
	return r
}
