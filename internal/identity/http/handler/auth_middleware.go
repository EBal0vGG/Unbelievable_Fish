package handler

import (
	"net/http"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type AuthMiddleware struct {
	tokens *identityauth.TokenProvider
}

func NewAuthMiddleware(tokens *identityauth.TokenProvider) *AuthMiddleware {
	return &AuthMiddleware{tokens: tokens}
}

func (m *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta := readMeta(r)
		identity, err := identityauth.ValidateBearerToken(m.tokens, r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, err, meta)
			return
		}
		next.ServeHTTP(w, r.WithContext(identityauth.WithIdentity(r.Context(), identity)))
	})
}

func (m *AuthMiddleware) RequireRole(role identity.Role, next http.Handler) http.Handler {
	return m.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !identityauth.HasRole(r.Context(), role) {
			writeError(w, identityauth.ErrForbidden, readMeta(r))
			return
		}
		next.ServeHTTP(w, r)
	}))
}
