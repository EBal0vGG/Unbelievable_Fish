package handler

import (
	"net/http"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

// AuthMiddleware is the identity service JWT gate; errors use identity httpapi.MapError.
type AuthMiddleware = identityauth.Middleware

// NewAuthMiddleware builds JWT middleware that emits JSON errors consistent with other identity handlers.
func NewAuthMiddleware(tokens *identityauth.TokenProvider) *identityauth.Middleware {
	return identityauth.NewMiddleware(tokens, func(w http.ResponseWriter, r *http.Request, err error) {
		writeError(w, err, readMeta(r))
	})
}
