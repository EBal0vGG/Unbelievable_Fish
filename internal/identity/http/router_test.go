package httpapi_test

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
	"github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http/handler"
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

func TestCommandFlowSmoke(t *testing.T) {
	companies := newFakeCompanyRepo()
	users := newFakeUserRepo()

	registerCompanyUC, err := identityapp.NewRegisterCompany(companies, fixedIDGenerator{companyID: "company-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	registerUserUC, err := identityapp.NewRegisterUser(users, companies, fakePasswordHasher{hashValue: "hashed:"}, fixedIDGenerator{userID: "user-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 5, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	tokenProvider := identityauth.NewTokenProvider("secret", time.Hour)
	loginUC, err := identityapp.NewLogin(users, fakePasswordHasher{hashValue: "hashed:", compareOK: true}, tokenProvider)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	getCurrentUserUC, err := identityapp.NewGetCurrentUser(users)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	router := httpapi.NewRouter(
		handler.NewRegisterCompanyHandler(registerCompanyUC),
		handler.NewRegisterUserHandler(registerUserUC),
		handler.NewLoginHandler(loginUC),
		handler.NewAuthMiddleware(tokenProvider).Wrap(handler.NewGetCurrentUserHandler(getCurrentUserUC)),
	)

	companyBody, _ := json.Marshal(httpapi.RegisterCompanyRequest{
		Name: "Acme Fish",
		INN:  "7707083893",
		OGRN: "1027700132195",
	})
	companyReq := httptest.NewRequest(http.MethodPost, "/companies", bytes.NewReader(companyBody))
	companyRec := httptest.NewRecorder()
	router.ServeHTTP(companyRec, companyReq)
	if companyRec.Code != http.StatusAccepted {
		t.Fatalf("expected register company status %d, got %d", http.StatusAccepted, companyRec.Code)
	}

	userBody, _ := json.Marshal(httpapi.RegisterUserRequest{
		CompanyID:     "company-1",
		Name:          "Alice",
		Role:          identity.RoleAdmin,
		Login:         "alice@example.com",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	userReq := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(userBody))
	userRec := httptest.NewRecorder()
	router.ServeHTTP(userRec, userReq)
	if userRec.Code != http.StatusAccepted {
		t.Fatalf("expected register user status %d, got %d", http.StatusAccepted, userRec.Code)
	}

	loginBody, _ := json.Marshal(httpapi.LoginRequest{
		Login:    "alice@example.com",
		Password: "secret",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginRec.Code)
	}
	var loginResp httpapi.LoginResponse
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	meRec := httptest.NewRecorder()
	router.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected current user status %d, got %d", http.StatusOK, meRec.Code)
	}
}

func TestCommandFlowSmokeWithoutCompany(t *testing.T) {
	companies := newFakeCompanyRepo()
	users := newFakeUserRepo()

	registerCompanyUC, err := identityapp.NewRegisterCompany(companies, fixedIDGenerator{companyID: "company-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	registerUserUC, err := identityapp.NewRegisterUser(users, companies, fakePasswordHasher{hashValue: "hashed:"}, fixedIDGenerator{userID: "user-2"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 5, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	tokenProvider := identityauth.NewTokenProvider("secret", time.Hour)
	loginUC, err := identityapp.NewLogin(users, fakePasswordHasher{hashValue: "hashed:", compareOK: true}, tokenProvider)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	getCurrentUserUC, err := identityapp.NewGetCurrentUser(users)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	router := httpapi.NewRouter(
		handler.NewRegisterCompanyHandler(registerCompanyUC),
		handler.NewRegisterUserHandler(registerUserUC),
		handler.NewLoginHandler(loginUC),
		handler.NewAuthMiddleware(tokenProvider).Wrap(handler.NewGetCurrentUserHandler(getCurrentUserUC)),
	)

	userBody, _ := json.Marshal(httpapi.RegisterUserRequest{
		Name:          "Bob",
		Role:          identity.RoleBuyer,
		Login:         "bob@example.com",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	userReq := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(userBody))
	userRec := httptest.NewRecorder()
	router.ServeHTTP(userRec, userReq)
	if userRec.Code != http.StatusAccepted {
		t.Fatalf("expected register user status %d, got %d", http.StatusAccepted, userRec.Code)
	}

	loginBody, _ := json.Marshal(httpapi.LoginRequest{
		Login:    "bob@example.com",
		Password: "secret",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d", http.StatusOK, loginRec.Code)
	}
	var loginResp httpapi.LoginResponse
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResp.User.CompanyID != "" {
		t.Fatalf("expected empty company id, got %q", loginResp.User.CompanyID)
	}
}
