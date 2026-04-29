package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(
	registerCompany http.Handler,
	registerUser http.Handler,
	listUsers http.Handler,
	promoteUserAdmin http.Handler,
	login http.Handler,
	getCurrentUser http.Handler,
) chi.Router {
	r := chi.NewRouter()
	r.Method(http.MethodPost, "/companies", registerCompany)
	r.Method(http.MethodPost, "/users", registerUser)
	r.Method(http.MethodGet, "/users", listUsers)
	r.Method(http.MethodPost, "/users/{userID}/promote-admin", promoteUserAdmin)
	r.Method(http.MethodPost, "/auth/login", login)
	r.Method(http.MethodGet, "/users/me", getCurrentUser)
	return r
}
