package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers groups HTTP handlers for the trading service.
type Handlers struct {
	ListAuctions   http.Handler
	PublishAuction http.Handler
	PlaceBid       http.Handler
	CloseAuction   http.Handler
	CancelAuction  http.Handler
	GetByID        http.Handler
	GetByLot       http.Handler
}

// NewRouter registers trading routes. Variadic middlewares apply to all routes.
func NewRouter(h Handlers, middlewares ...func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares...)

	r.Route("/auctions", func(r chi.Router) {
		r.Method(http.MethodGet, "/", h.ListAuctions)
		r.Method(http.MethodGet, "/by-lot/{lotID}", h.GetByLot)
		r.Route("/{auctionID}", func(r chi.Router) {
			r.Method(http.MethodGet, "/", h.GetByID)
			r.Method(http.MethodPost, "/publish", h.PublishAuction)
			r.Method(http.MethodPost, "/bids", h.PlaceBid)
			r.Method(http.MethodPost, "/close", h.CloseAuction)
			r.Method(http.MethodPost, "/cancel", h.CancelAuction)
		})
	})

	return r
}
