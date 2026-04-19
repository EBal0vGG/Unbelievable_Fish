package auth

import (
	"context"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type Identity struct {
	UserID    string
	CompanyID string
	Role      identity.Role
}

type identityContextKey struct{}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityContextKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityContextKey{}).(Identity)
	if !ok || identity.UserID == "" {
		return Identity{}, false
	}
	return identity, true
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return "", false
	}
	return identity.UserID, true
}

func CompanyIDFromContext(ctx context.Context) (string, bool) {
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return "", false
	}
	return identity.CompanyID, true
}

func RoleFromContext(ctx context.Context) (identity.Role, bool) {
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return "", false
	}
	return identity.Role, true
}

func HasRole(ctx context.Context, role identity.Role) bool {
	current, ok := RoleFromContext(ctx)
	if !ok {
		return false
	}
	return current == role
}
