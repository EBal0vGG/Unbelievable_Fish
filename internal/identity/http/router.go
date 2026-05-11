package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handlers groups HTTP handlers for the identity service.
type Handlers struct {
	RegisterCompany    http.Handler
	RegisterUser       http.Handler
	ListUsers          http.Handler
	PromoteUserAdmin   http.Handler
	Login              http.Handler
	VerifyEmail        http.Handler
	ResendVerification http.Handler
	GetCurrentUser     http.Handler
}

// NewRouter registers identity routes. Variadic middlewares apply to all routes.
func NewRouter(h Handlers, middlewares ...func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares...)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"identity"}`))
	})

	r.Route("/companies", func(r chi.Router) {
		r.Method(http.MethodPost, "/", h.RegisterCompany)
	})

	r.Route("/auth", func(r chi.Router) {
		r.Method(http.MethodPost, "/login", h.Login)
		r.Method(http.MethodGet, "/verify-email", handlerOrNotFound(h.VerifyEmail))
		r.Method(http.MethodPost, "/verify-email", handlerOrNotFound(h.VerifyEmail))
		r.Method(http.MethodPost, "/resend-verification", handlerOrNotFound(h.ResendVerification))
	})

	r.Route("/users", func(r chi.Router) {
		r.Method(http.MethodGet, "/me", h.GetCurrentUser)
		r.Method(http.MethodPost, "/", h.RegisterUser)
		r.Method(http.MethodGet, "/", h.ListUsers)
		r.Method(http.MethodPost, "/{userID}/promote-admin", h.PromoteUserAdmin)
	})

	return r
}

func handlerOrNotFound(h http.Handler) http.Handler {
	if h == nil {
		return http.NotFoundHandler()
	}
	return h
}
