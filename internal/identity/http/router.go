package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers groups HTTP handlers for the identity service.
type Handlers struct {
	RegisterCompany  http.Handler
	RegisterUser     http.Handler
	ListUsers        http.Handler
	PromoteUserAdmin http.Handler
	Login            http.Handler
	GetCurrentUser   http.Handler
}

// NewRouter registers identity routes. Variadic middlewares apply to all routes.
func NewRouter(h Handlers, middlewares ...func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares...)

	r.Route("/companies", func(r chi.Router) {
		r.Method(http.MethodPost, "/", h.RegisterCompany)
	})

	r.Route("/auth", func(r chi.Router) {
		r.Method(http.MethodPost, "/login", h.Login)
	})

	r.Route("/users", func(r chi.Router) {
		r.Method(http.MethodGet, "/me", h.GetCurrentUser)
		r.Method(http.MethodPost, "/", h.RegisterUser)
		r.Method(http.MethodGet, "/", h.ListUsers)
		r.Method(http.MethodPost, "/{userID}/promote-admin", h.PromoteUserAdmin)
	})

	return r
}
