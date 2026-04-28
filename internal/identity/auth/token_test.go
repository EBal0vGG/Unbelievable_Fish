package auth

import (
	"context"
	"testing"
	"time"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

func TestTokenGenerationAndValidation(t *testing.T) {
	provider := NewTokenProvider("secret", time.Hour)
	provider.now = func() time.Time {
		return time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)
	}

	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleAdmin, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}

	token, err := provider.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if token == "" {
		t.Fatal("expected token to be generated")
	}

	claims, err := provider.Validate(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.UserID != "user-1" || claims.CompanyID != "company-1" || claims.Role != identity.RoleAdmin {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestTokenGenerationAndValidationWithoutCompany(t *testing.T) {
	provider := NewTokenProvider("secret", time.Hour)
	provider.now = func() time.Time {
		return time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)
	}

	user, err := identity.NewUser("user-2", "", "Bob", identity.RoleBuyer, "bob@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}

	token, err := provider.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := provider.Validate(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.UserID != "user-2" || claims.CompanyID != "" || claims.Role != identity.RoleBuyer {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestTokenValidationInvalidToken(t *testing.T) {
	provider := NewTokenProvider("secret", time.Hour)
	if _, err := provider.Validate("invalid.token"); err != ErrInvalidToken {
		t.Fatalf("expected %v, got %v", ErrInvalidToken, err)
	}
}

func TestContextHelpers(t *testing.T) {
	ctx := WithIdentity(context.Background(), Identity{
		UserID:    "user-1",
		CompanyID: "company-1",
		Role:      identity.RoleSeller,
	})

	userID, ok := UserIDFromContext(ctx)
	if !ok || userID != "user-1" {
		t.Fatalf("expected user id user-1, got %q %v", userID, ok)
	}
	companyID, ok := CompanyIDFromContext(ctx)
	if !ok || companyID != "company-1" {
		t.Fatalf("expected company id company-1, got %q %v", companyID, ok)
	}
	role, ok := RoleFromContext(ctx)
	if !ok || role != identity.RoleSeller {
		t.Fatalf("expected role seller, got %q %v", role, ok)
	}
	if !HasRole(ctx, identity.RoleSeller) {
		t.Fatal("expected seller role to match")
	}
}
