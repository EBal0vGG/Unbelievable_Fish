package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers groups HTTP handlers for the deals service (one field per route).
type Handlers struct {
	GetDealProjection   http.Handler
	GetDealByAuction    http.Handler
	GetDeal             http.Handler
	GetConfirmations    http.Handler
	RequestConfirmation http.Handler
	ApproveConfirmation http.Handler
	RejectConfirmation  http.Handler
	PrepareContract     http.Handler
	SignContract        http.Handler
	RequestPayment      http.Handler
	RequestShipment     http.Handler
	UpdateDealPrice     http.Handler
}

// NewRouter registers deals routes. Variadic middlewares apply to all routes (e.g. logging, auth).
func NewRouter(h Handlers, middlewares ...func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares...)

	r.Route("/deal-projections", func(r chi.Router) {
		r.Method(http.MethodGet, "/{auctionID}", h.GetDealProjection)
	})

	r.Route("/deals", func(r chi.Router) {
		r.Method(http.MethodGet, "/by-auction/{auctionID}", h.GetDealByAuction)
		r.Route("/{dealID}", func(r chi.Router) {
			r.Method(http.MethodGet, "/", h.GetDeal)
			r.Method(http.MethodGet, "/confirmations", h.GetConfirmations)
			r.Method(http.MethodPost, "/confirmations", h.RequestConfirmation)
			r.Method(http.MethodPost, "/confirmations/{confirmationID}/approve", h.ApproveConfirmation)
			r.Method(http.MethodPost, "/confirmations/{confirmationID}/reject", h.RejectConfirmation)
			r.Method(http.MethodPost, "/contract/prepare", h.PrepareContract)
			r.Method(http.MethodPost, "/contract/sign", h.SignContract)
			r.Method(http.MethodPost, "/payment/request", h.RequestPayment)
			r.Method(http.MethodPost, "/shipment/request", h.RequestShipment)
			r.Method(http.MethodPost, "/price", h.UpdateDealPrice)
		})
	})

	return r
}
