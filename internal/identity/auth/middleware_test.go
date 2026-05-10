package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

func TestMiddlewareSuccessWithValidToken(t *testing.T) {
	provider := NewTokenProvider("secret", time.Hour)
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleBuyer, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}
	token, err := provider.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	middleware := NewMiddleware(provider, nil)
	next := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := IdentityFromContext(r.Context())
		_ = json.NewEncoder(w).Encode(id)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	var got Identity
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.UserID != "user-1" || got.CompanyID != "company-1" || got.Role != identity.RoleBuyer {
		t.Fatalf("unexpected identity: %+v", got)
	}
}

func TestMiddlewareRejectWithoutToken(t *testing.T) {
	called := false
	middleware := NewMiddleware(NewTokenProvider("secret", time.Hour), func(w http.ResponseWriter, r *http.Request, err error) {
		called = true
		http.Error(w, err.Error(), http.StatusUnauthorized)
	})
	next := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected error handler to be called")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestMiddlewareRequireRoleForbidden(t *testing.T) {
	provider := NewTokenProvider("secret", time.Hour)
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleBuyer, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}
	token, err := provider.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	middleware := NewMiddleware(provider, func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, err.Error(), http.StatusForbidden)
	})
	next := middleware.RequireRole(identity.RoleSeller, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestMiddlewareRequireRoleAllowsBuyerSellerRole(t *testing.T) {
	provider := NewTokenProvider("secret", time.Hour)
	user, err := identity.NewUser("user-2", "company-1", "Bob", identity.RoleBuyerSeller, "bob@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}
	token, err := provider.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	middleware := NewMiddleware(provider, func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, err.Error(), http.StatusForbidden)
	})
	next := middleware.RequireRole(identity.RoleSeller, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestMiddlewareRequireOneOfRolesAllowsSeller(t *testing.T) {
	provider := NewTokenProvider("secret", time.Hour)
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleSeller, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}
	token, err := provider.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	middleware := NewMiddleware(provider, func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, err.Error(), http.StatusForbidden)
	})
	next := middleware.RequireOneOfRoles(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), identity.RoleSeller, identity.RoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestMiddlewareRequireOneOfRolesForbiddenForBuyer(t *testing.T) {
	provider := NewTokenProvider("secret", time.Hour)
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleBuyer, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}
	token, err := provider.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	middleware := NewMiddleware(provider, func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, err.Error(), http.StatusForbidden)
	})
	next := middleware.RequireOneOfRoles(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), identity.RoleSeller, identity.RoleAdmin)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}
