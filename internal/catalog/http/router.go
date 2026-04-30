package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRouter registers catalog HTTP routes. Protected lot routes should be wrapped
// with auth before being passed in. Variadic middlewares typically include httplog only.
func NewRouter(
	listFish http.Handler,
	createFish http.Handler,
	createProduct http.Handler,
	publishProduct http.Handler,
	createLot http.Handler,
	publishLot http.Handler,
	middlewares ...func(http.Handler) http.Handler,
) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares...)
	r.Method(http.MethodGet, "/fish", listFish)
	r.Method(http.MethodPost, "/fish", createFish)
	r.Method(http.MethodPost, "/products", createProduct)
	r.Method(http.MethodPost, "/products/{productID}/publish", publishProduct)
	r.Method(http.MethodPost, "/lots", createLot)
	r.Method(http.MethodPost, "/lots/{lotID}/publish", publishLot)
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r
}
