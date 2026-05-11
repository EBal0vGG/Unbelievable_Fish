package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
)

type fakeUserRepo struct {
	byID    map[string]*identity.User
	byLogin map[string]*identity.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:    make(map[string]*identity.User),
		byLogin: make(map[string]*identity.User),
	}
}

func (r *fakeUserRepo) Save(ctx context.Context, user *identity.User) error {
	_ = ctx
	r.byID[user.ID()] = user
	r.byLogin[user.Login()] = user
	return nil
}

func (r *fakeUserRepo) GetByID(ctx context.Context, userID string) (*identity.User, error) {
	_ = ctx
	user, ok := r.byID[userID]
	if !ok {
		return nil, identityapp.ErrUserNotFound
	}
	return user, nil
}

func (r *fakeUserRepo) GetByLogin(ctx context.Context, login string) (*identity.User, error) {
	_ = ctx
	user, ok := r.byLogin[login]
	if !ok {
		return nil, identityapp.ErrUserNotFound
	}
	return user, nil
}

func (r *fakeUserRepo) List(ctx context.Context) ([]*identity.User, error) {
	_ = ctx
	users := make([]*identity.User, 0, len(r.byID))
	for _, user := range r.byID {
		users = append(users, user)
	}
	return users, nil
}

type fakeCompanyRepo struct {
	byID  map[string]*identity.Company
	byKey map[string]*identity.Company
}

func newFakeCompanyRepo() *fakeCompanyRepo {
	return &fakeCompanyRepo{
		byID:  make(map[string]*identity.Company),
		byKey: make(map[string]*identity.Company),
	}
}

func (r *fakeCompanyRepo) Save(ctx context.Context, company *identity.Company) error {
	_ = ctx
	r.byID[company.ID()] = company
	r.byKey[company.INN()+"|"+company.OGRN()] = company
	return nil
}

func (r *fakeCompanyRepo) GetByID(ctx context.Context, companyID string) (*identity.Company, error) {
	_ = ctx
	company, ok := r.byID[companyID]
	if !ok {
		return nil, identityapp.ErrCompanyNotFound
	}
	return company, nil
}

func (r *fakeCompanyRepo) GetByRequisites(ctx context.Context, inn string, ogrn string) (*identity.Company, error) {
	_ = ctx
	company, ok := r.byKey[inn+"|"+ogrn]
	if !ok {
		return nil, identityapp.ErrCompanyNotFound
	}
	return company, nil
}

type fakePasswordHasher struct {
	hashValue string
	compareOK bool
}

func (h fakePasswordHasher) Hash(password string) (string, error) {
	return h.hashValue + password, nil
}

func (h fakePasswordHasher) Compare(passwordHash, password string) (bool, error) {
	return h.compareOK && passwordHash == h.hashValue+password, nil
}

type fakeTokenProvider struct {
	token string
}

func (p fakeTokenProvider) Generate(user *identity.User) (string, error) {
	_ = user
	return p.token, nil
}

type fixedIDGenerator struct {
	companyID string
	userID    string
}

func (g fixedIDGenerator) NewCompanyID() string { return g.companyID }
func (g fixedIDGenerator) NewUserID() string    { return g.userID }

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time { return c.now }

func TestRegisterCompanyHandlerSuccess(t *testing.T) {
	companies := newFakeCompanyRepo()
	uc, err := identityapp.NewRegisterCompany(companies, fixedIDGenerator{companyID: "company-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewRegisterCompanyHandler(uc)

	body, _ := json.Marshal(httpapi.RegisterCompanyRequest{
		Name: "Acme Fish",
		INN:  "7707083893",
		OGRN: "1027700132195",
	})
	req := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	var resp httpapi.CompanyResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ID != "company-1" {
		t.Fatalf("expected company id company-1, got %q", resp.ID)
	}
}

func TestRegisterUserHandlerSuccess(t *testing.T) {
	companies := newFakeCompanyRepo()
	company, err := identity.NewCompany("company-1", "Acme Fish", "7707083893", "1027700132195", time.Now())
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	companies.byID[company.ID()] = company

	users := newFakeUserRepo()
	uc, err := identityapp.NewRegisterUser(users, companies, fakePasswordHasher{hashValue: "hashed:"}, fixedIDGenerator{userID: "user-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 5, 0, 0, time.UTC)}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewRegisterUserHandler(uc)

	body, _ := json.Marshal(httpapi.RegisterUserRequest{
		CompanyID:     "company-1",
		Name:          "Alice",
		Role:          identity.RoleSeller,
		Login:         "alice@example.com",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	var resp httpapi.UserResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ID != "user-1" {
		t.Fatalf("expected user id user-1, got %q", resp.ID)
	}
}

func TestRegisterUserHandlerSuccessWithBuyerSellerRole(t *testing.T) {
	companies := newFakeCompanyRepo()
	company, err := identity.NewCompany("company-1", "Acme Fish", "7707083893", "1027700132195", time.Now())
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	companies.byID[company.ID()] = company

	users := newFakeUserRepo()
	uc, err := identityapp.NewRegisterUser(users, companies, fakePasswordHasher{hashValue: "hashed:"}, fixedIDGenerator{userID: "user-1b"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 6, 0, 0, time.UTC)}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewRegisterUserHandler(uc)

	body, _ := json.Marshal(httpapi.RegisterUserRequest{
		CompanyID:     "company-1",
		Name:          "Alice",
		Role:          identity.RoleBuyerSeller,
		Login:         "alice-both@example.com",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	var resp httpapi.UserResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Role != identity.RoleBuyerSeller {
		t.Fatalf("expected role %q, got %q", identity.RoleBuyerSeller, resp.Role)
	}
}

func TestRegisterUserHandlerSuccessWithoutCompany(t *testing.T) {
	companies := newFakeCompanyRepo()
	users := newFakeUserRepo()
	uc, err := identityapp.NewRegisterUser(users, companies, fakePasswordHasher{hashValue: "hashed:"}, fixedIDGenerator{companyID: "company-shell-bob", userID: "user-2"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 5, 0, 0, time.UTC)}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewRegisterUserHandler(uc)

	body, _ := json.Marshal(httpapi.RegisterUserRequest{
		Name:          "Bob",
		Role:          identity.RoleBuyer,
		Login:         "bob@example.com",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	var resp httpapi.UserResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.CompanyID != "company-shell-bob" {
		t.Fatalf("expected shell company id company-shell-bob, got %q", resp.CompanyID)
	}
}

func TestRegisterUserHandlerRequiresTermsAcceptance(t *testing.T) {
	companies := newFakeCompanyRepo()
	company, err := identity.NewCompany("company-1", "Acme Fish", "7707083893", "1027700132195", time.Now())
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	companies.byID[company.ID()] = company

	users := newFakeUserRepo()
	uc, err := identityapp.NewRegisterUser(users, companies, fakePasswordHasher{hashValue: "hashed:"}, fixedIDGenerator{userID: "user-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 5, 0, 0, time.UTC)}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewRegisterUserHandler(uc)

	body, _ := json.Marshal(httpapi.RegisterUserRequest{
		CompanyID:     "company-1",
		Name:          "Alice",
		Role:          identity.RoleSeller,
		Login:         "alice@example.com",
		Password:      "secret",
		AcceptedTerms: false,
		TermsVersion:  "2026-04-24",
	})
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	assertErrorCode(t, rec, "TERMS_ACCEPTANCE_REQUIRED")
}

func TestLoginHandlerSuccess(t *testing.T) {
	users := newFakeUserRepo()
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleAdmin, "alice@example.com", "hashed:secret")
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	user.VerifyEmail()
	users.byID[user.ID()] = user
	users.byLogin[user.Login()] = user

	uc, err := identityapp.NewLogin(users, fakePasswordHasher{hashValue: "hashed:", compareOK: true}, fakeTokenProvider{token: "token-1"})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewLoginHandler(uc)

	body, _ := json.Marshal(httpapi.LoginRequest{
		Login:    "alice@example.com",
		Password: "secret",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var resp httpapi.LoginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Token != "token-1" {
		t.Fatalf("expected token token-1, got %q", resp.Token)
	}
}

func TestLoginHandlerInvalidCredentials(t *testing.T) {
	users := newFakeUserRepo()
	uc, err := identityapp.NewLogin(users, fakePasswordHasher{hashValue: "hashed:", compareOK: true}, fakeTokenProvider{token: "token-1"})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewLoginHandler(uc)

	body, _ := json.Marshal(httpapi.LoginRequest{
		Login:    "missing@example.com",
		Password: "secret",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	assertErrorCode(t, rec, "INVALID_CREDENTIALS")
}

func TestGetCurrentUserHandlerSuccess(t *testing.T) {
	users := newFakeUserRepo()
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleAdmin, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	users.byID[user.ID()] = user
	users.byLogin[user.Login()] = user

	uc, err := identityapp.NewGetCurrentUser(users)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewGetCurrentUserHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = req.WithContext(identityauth.WithIdentity(req.Context(), identityauth.Identity{
		UserID:    "user-1",
		CompanyID: "company-1",
		Role:      identity.RoleAdmin,
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	var resp httpapi.UserResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ID != "user-1" {
		t.Fatalf("expected user id user-1, got %q", resp.ID)
	}
}

func TestGetCurrentUserHandlerError(t *testing.T) {
	users := newFakeUserRepo()
	uc, err := identityapp.NewGetCurrentUser(users)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewGetCurrentUserHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = req.WithContext(identityauth.WithIdentity(req.Context(), identityauth.Identity{
		UserID:    "missing",
		CompanyID: "company-1",
		Role:      identity.RoleAdmin,
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	assertErrorCode(t, rec, "USER_NOT_FOUND")
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var resp httpapi.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if resp.Code != want {
		t.Fatalf("expected error code %s, got %s", want, resp.Code)
	}
}
