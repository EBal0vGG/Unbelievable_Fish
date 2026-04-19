package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
)

func TestAuthMiddlewareSuccess(t *testing.T) {
	tokens := NewTestTokenProvider(t)
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleAdmin, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}
	token, err := tokens.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	middleware := NewAuthMiddleware(tokens)
	next := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := identityauth.UserIDFromContext(r.Context())
		companyID, _ := identityauth.CompanyIDFromContext(r.Context())
		role, _ := identityauth.RoleFromContext(r.Context())
		_ = json.NewEncoder(w).Encode(map[string]string{
			"user_id":    userID,
			"company_id": companyID,
			"role":       string(role),
		})
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var payload map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if payload["user_id"] != "user-1" || payload["company_id"] != "company-1" || payload["role"] != "admin" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestAuthMiddlewareRejectWithoutToken(t *testing.T) {
	middleware := NewAuthMiddleware(NewTestTokenProvider(t))
	next := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	assertMiddlewareErrorCode(t, rec, "MISSING_AUTHORIZATION")
}

func TestAuthMiddlewareRejectInvalidToken(t *testing.T) {
	middleware := NewAuthMiddleware(NewTestTokenProvider(t))
	next := middleware.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.value")
	rec := httptest.NewRecorder()

	next.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	assertMiddlewareErrorCode(t, rec, "INVALID_TOKEN")
}

func assertMiddlewareErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var resp httpapi.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if resp.Code != want {
		t.Fatalf("expected error code %s, got %s", want, resp.Code)
	}
}

func NewTestTokenProvider(t *testing.T) *identityauth.TokenProvider {
	t.Helper()
	return identityauth.NewTokenProvider("secret", time.Hour)
}
