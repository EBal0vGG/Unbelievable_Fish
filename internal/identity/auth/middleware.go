package auth

import (
	"net/http"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type ErrorHandler func(http.ResponseWriter, *http.Request, error)

type Middleware struct {
	tokens  *TokenProvider
	onError ErrorHandler
}

func NewMiddleware(tokens *TokenProvider, onError ErrorHandler) *Middleware {
	return &Middleware{
		tokens:  tokens,
		onError: onError,
	}
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, err := ValidateBearerToken(m.tokens, r.Header.Get("Authorization"))
		if err != nil {
			m.handleError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), identity)))
	})
}

func (m *Middleware) RequireRole(role identity.Role, next http.Handler) http.Handler {
	return m.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !HasRole(r.Context(), role) {
			m.handleError(w, r, ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// RequireOneOfRoles allows the request when the JWT role matches any of the given roles
// (using identity.IncludesRole, so buyer_seller satisfies buyer or seller requirements).
func (m *Middleware) RequireOneOfRoles(next http.Handler, roles ...identity.Role) http.Handler {
	return m.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current, ok := RoleFromContext(r.Context())
		if !ok {
			m.handleError(w, r, ErrForbidden)
			return
		}
		for _, role := range roles {
			if identity.IncludesRole(current, role) {
				next.ServeHTTP(w, r)
				return
			}
		}
		m.handleError(w, r, ErrForbidden)
	}))
}

func (m *Middleware) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if m.onError != nil {
		m.onError(w, r, err)
		return
	}
	http.Error(w, err.Error(), http.StatusUnauthorized)
}
