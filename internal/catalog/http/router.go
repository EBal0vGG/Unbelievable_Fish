package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers groups HTTP handlers for the catalog service.
type Handlers struct {
	ListFish       http.Handler
	CreateFish     http.Handler
	CreateProduct  http.Handler
	PublishProduct http.Handler
	ListProducts   http.Handler
	CreateLot      http.Handler
	PublishLot     http.Handler
	ListLots       http.Handler
}

// NewRouter registers catalog HTTP routes. Protected lot routes should be wrapped
// with auth before being passed in. Variadic middlewares typically include httplog only.
func NewRouter(h Handlers, middlewares ...func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares...)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/fish", func(r chi.Router) {
		r.Method(http.MethodGet, "/", h.ListFish)
		r.Method(http.MethodPost, "/", h.CreateFish)
	})

	r.Route("/products", func(r chi.Router) {
		r.Method(http.MethodGet, "/", h.ListProducts)
		r.Method(http.MethodPost, "/", h.CreateProduct)
		r.Method(http.MethodPost, "/{productID}/publish", h.PublishProduct)
	})

	r.Route("/lots", func(r chi.Router) {
		r.Method(http.MethodGet, "/", h.ListLots)
		r.Method(http.MethodPost, "/", h.CreateLot)
		r.Method(http.MethodPost, "/{lotID}/publish", h.PublishLot)
	})

	return r
}
