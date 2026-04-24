package app

import (
	"context"
	"strings"
	"testing"
	"time"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type fakeUserRepo struct {
	byID        map[string]*identity.User
	byLogin     map[string]*identity.User
	saveErr     error
	getByIDErr  error
	getByLogErr error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:    make(map[string]*identity.User),
		byLogin: make(map[string]*identity.User),
	}
}

func (r *fakeUserRepo) Save(ctx context.Context, user *identity.User) error {
	_ = ctx
	if r.saveErr != nil {
		return r.saveErr
	}
	r.byID[user.ID()] = user
	r.byLogin[user.Login()] = user
	return nil
}

func (r *fakeUserRepo) GetByID(ctx context.Context, userID string) (*identity.User, error) {
	_ = ctx
	if r.getByIDErr != nil {
		return nil, r.getByIDErr
	}
	user, ok := r.byID[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (r *fakeUserRepo) GetByLogin(ctx context.Context, login string) (*identity.User, error) {
	_ = ctx
	if r.getByLogErr != nil {
		return nil, r.getByLogErr
	}
	user, ok := r.byLogin[strings.ToLower(strings.TrimSpace(login))]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

type fakeCompanyRepo struct {
	byID       map[string]*identity.Company
	byKey      map[string]*identity.Company
	saveErr    error
	getByIDErr error
}

func newFakeCompanyRepo() *fakeCompanyRepo {
	return &fakeCompanyRepo{
		byID:  make(map[string]*identity.Company),
		byKey: make(map[string]*identity.Company),
	}
}

func (r *fakeCompanyRepo) Save(ctx context.Context, company *identity.Company) error {
	_ = ctx
	if r.saveErr != nil {
		return r.saveErr
	}
	r.byID[company.ID()] = company
	r.byKey[company.INN()+"|"+company.OGRN()] = company
	return nil
}

func (r *fakeCompanyRepo) GetByID(ctx context.Context, companyID string) (*identity.Company, error) {
	_ = ctx
	if r.getByIDErr != nil {
		return nil, r.getByIDErr
	}
	company, ok := r.byID[companyID]
	if !ok {
		return nil, ErrCompanyNotFound
	}
	return company, nil
}

func (r *fakeCompanyRepo) GetByRequisites(ctx context.Context, inn string, ogrn string) (*identity.Company, error) {
	_ = ctx
	company, ok := r.byKey[strings.TrimSpace(inn)+"|"+strings.TrimSpace(ogrn)]
	if !ok {
		return nil, ErrCompanyNotFound
	}
	return company, nil
}

type fakePasswordHasher struct {
	hashValue   string
	hashErr     error
	compareOK   bool
	compareErr  error
	lastHashed  string
	lastHash    string
	lastCompare string
}

func (h *fakePasswordHasher) Hash(password string) (string, error) {
	h.lastHashed = password
	if h.hashErr != nil {
		return "", h.hashErr
	}
	if h.hashValue != "" {
		return h.hashValue, nil
	}
	return "hashed:" + password, nil
}

func (h *fakePasswordHasher) Compare(passwordHash, password string) (bool, error) {
	h.lastHash = passwordHash
	h.lastCompare = password
	if h.compareErr != nil {
		return false, h.compareErr
	}
	return h.compareOK, nil
}

type fakeTokenProvider struct {
	token    string
	err      error
	lastUser *identity.User
}

func (p *fakeTokenProvider) Generate(user *identity.User) (string, error) {
	p.lastUser = user
	if p.err != nil {
		return "", p.err
	}
	if p.token != "" {
		return p.token, nil
	}
	return "token", nil
}

type fakeIDGenerator struct {
	companyID string
	userID    string
}

func (g fakeIDGenerator) NewCompanyID() string {
	return g.companyID
}

func (g fakeIDGenerator) NewUserID() string {
	return g.userID
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestRegisterCompanySuccess(t *testing.T) {
	companies := newFakeCompanyRepo()
	now := time.Date(2024, time.March, 1, 10, 0, 0, 0, time.UTC)

	uc, err := NewRegisterCompany(companies, fakeIDGenerator{companyID: "company-1"}, fixedClock{now: now})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	result, err := uc.Execute(context.Background(), RegisterCompanyCommand{
		Name: "Acme Fish",
		INN:  "7707083893",
		OGRN: "1027700132195",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "company-1" {
		t.Fatalf("expected company id company-1, got %q", result.ID)
	}
	if result.Status != identity.CompanyStatusActive {
		t.Fatalf("expected active status, got %q", result.Status)
	}
	if !result.CreatedAt.Equal(now) {
		t.Fatalf("expected created at %v, got %v", now, result.CreatedAt)
	}
	if _, err := companies.GetByID(context.Background(), "company-1"); err != nil {
		t.Fatalf("expected company to be stored, got %v", err)
	}
}

func TestRegisterCompanyReturnsExistingCompanyByRequisites(t *testing.T) {
	companies := newFakeCompanyRepo()
	existing, err := identity.NewCompany(
		"company-existing",
		"North Sea LLC",
		"7707083893",
		"1027700132195",
		time.Date(2024, time.March, 1, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	companies.byID[existing.ID()] = existing
	companies.byKey[existing.INN()+"|"+existing.OGRN()] = existing

	uc, err := NewRegisterCompany(companies, fakeIDGenerator{companyID: "company-new"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	result, err := uc.Execute(context.Background(), RegisterCompanyCommand{
		Name: "Other Name",
		INN:  "7707083893",
		OGRN: "1027700132195",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "company-existing" {
		t.Fatalf("expected existing company id, got %q", result.ID)
	}
	if _, ok := companies.byID["company-new"]; ok {
		t.Fatalf("expected no new company to be created")
	}
}

func TestRegisterUserSuccess(t *testing.T) {
	companies := newFakeCompanyRepo()
	company, err := identity.NewCompany("company-1", "Acme Fish", "7707083893", "1027700132195", time.Now())
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	companies.byID[company.ID()] = company
	companies.byKey[company.INN()+"|"+company.OGRN()] = company
	users := newFakeUserRepo()
	hasher := &fakePasswordHasher{hashValue: "hashed-password"}
	acceptedAt := time.Date(2024, time.April, 1, 11, 30, 0, 0, time.UTC)

	uc, err := NewRegisterUser(users, companies, hasher, fakeIDGenerator{userID: "user-1"}, fixedClock{now: acceptedAt})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	result, err := uc.Execute(context.Background(), RegisterUserCommand{
		CompanyID:     "company-1",
		Name:          "Alice",
		Role:          identity.RoleAdmin,
		Login:         " Alice@Example.com ",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "user-1" {
		t.Fatalf("expected user id user-1, got %q", result.ID)
	}
	if result.Login != "alice@example.com" {
		t.Fatalf("expected normalized login, got %q", result.Login)
	}
	if hasher.lastHashed != "secret" {
		t.Fatalf("expected password to be hashed, got %q", hasher.lastHashed)
	}
	stored, err := users.GetByID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected stored user, got %v", err)
	}
	if stored.PasswordHash() != "hashed-password" {
		t.Fatalf("expected stored hash, got %q", stored.PasswordHash())
	}
	if stored.TermsVersion() != "2026-04-24" {
		t.Fatalf("expected stored terms version, got %q", stored.TermsVersion())
	}
	if !stored.TermsAcceptedAt().Equal(acceptedAt) {
		t.Fatalf("expected stored accepted at %v, got %v", acceptedAt, stored.TermsAcceptedAt())
	}
}

func TestRegisterUserSuccessByCompanyRequisites(t *testing.T) {
	companies := newFakeCompanyRepo()
	company, err := identity.NewCompany("company-1", "Acme Fish", "7707083893", "1027700132195", time.Now())
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	companies.byID[company.ID()] = company
	companies.byKey[company.INN()+"|"+company.OGRN()] = company
	users := newFakeUserRepo()
	hasher := &fakePasswordHasher{hashValue: "hashed-password"}

	uc, err := NewRegisterUser(users, companies, hasher, fakeIDGenerator{userID: "user-2"}, fixedClock{now: time.Date(2024, time.April, 2, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	result, err := uc.Execute(context.Background(), RegisterUserCommand{
		CompanyINN:    "7707083893",
		CompanyOGRN:   "1027700132195",
		Name:          "Bob",
		Role:          identity.RoleSeller,
		Login:         "bob@example.com",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "user-2" {
		t.Fatalf("expected user id user-2, got %q", result.ID)
	}
	if result.CompanyID != "company-1" {
		t.Fatalf("expected company id company-1, got %q", result.CompanyID)
	}
}

func TestRegisterUserFailsWhenCompanyNotFound(t *testing.T) {
	users := newFakeUserRepo()
	companies := newFakeCompanyRepo()
	hasher := &fakePasswordHasher{}

	uc, err := NewRegisterUser(users, companies, hasher, fakeIDGenerator{userID: "user-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	_, err = uc.Execute(context.Background(), RegisterUserCommand{
		CompanyID:     "missing-company",
		Name:          "Alice",
		Role:          identity.RoleAdmin,
		Login:         "alice@example.com",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	if err != ErrCompanyNotFound {
		t.Fatalf("expected %v, got %v", ErrCompanyNotFound, err)
	}
}

func TestRegisterUserFailsWhenLoginAlreadyUsed(t *testing.T) {
	companies := newFakeCompanyRepo()
	company, err := identity.NewCompany("company-1", "Acme Fish", "7707083893", "1027700132195", time.Now())
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	companies.byID[company.ID()] = company
	companies.byKey[company.INN()+"|"+company.OGRN()] = company

	users := newFakeUserRepo()
	existing, err := identity.NewUser("user-existing", "company-1", "Bob", identity.RoleSeller, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	users.byID[existing.ID()] = existing
	users.byLogin[existing.Login()] = existing

	uc, err := NewRegisterUser(users, companies, &fakePasswordHasher{}, fakeIDGenerator{userID: "user-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	_, err = uc.Execute(context.Background(), RegisterUserCommand{
		CompanyID:     "company-1",
		Name:          "Alice",
		Role:          identity.RoleAdmin,
		Login:         "Alice@Example.com",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	if err != ErrLoginAlreadyUsed {
		t.Fatalf("expected %v, got %v", ErrLoginAlreadyUsed, err)
	}
}

func TestRegisterUserFailsWhenPasswordEmpty(t *testing.T) {
	users := newFakeUserRepo()
	companies := newFakeCompanyRepo()
	hasher := &fakePasswordHasher{}

	uc, err := NewRegisterUser(users, companies, hasher, fakeIDGenerator{userID: "user-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	_, err = uc.Execute(context.Background(), RegisterUserCommand{
		CompanyID:     "company-1",
		Name:          "Alice",
		Role:          identity.RoleAdmin,
		Login:         "alice@example.com",
		Password:      " ",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	if err != ErrPasswordRequired {
		t.Fatalf("expected %v, got %v", ErrPasswordRequired, err)
	}
}

func TestRegisterUserFailsWhenTermsNotAccepted(t *testing.T) {
	users := newFakeUserRepo()
	companies := newFakeCompanyRepo()
	hasher := &fakePasswordHasher{}

	uc, err := NewRegisterUser(users, companies, hasher, fakeIDGenerator{userID: "user-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	_, err = uc.Execute(context.Background(), RegisterUserCommand{
		CompanyID:     "company-1",
		Name:          "Alice",
		Role:          identity.RoleAdmin,
		Login:         "alice@example.com",
		Password:      "secret",
		AcceptedTerms: false,
		TermsVersion:  "2026-04-24",
	})
	if err != ErrTermsAcceptanceRequired {
		t.Fatalf("expected %v, got %v", ErrTermsAcceptanceRequired, err)
	}
}

func TestRegisterUserFailsWhenTermsVersionEmpty(t *testing.T) {
	users := newFakeUserRepo()
	companies := newFakeCompanyRepo()
	hasher := &fakePasswordHasher{}

	uc, err := NewRegisterUser(users, companies, hasher, fakeIDGenerator{userID: "user-1"}, fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	_, err = uc.Execute(context.Background(), RegisterUserCommand{
		CompanyID:     "company-1",
		Name:          "Alice",
		Role:          identity.RoleAdmin,
		Login:         "alice@example.com",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  " ",
	})
	if err != ErrTermsVersionRequired {
		t.Fatalf("expected %v, got %v", ErrTermsVersionRequired, err)
	}
}

func TestLoginSuccess(t *testing.T) {
	users := newFakeUserRepo()
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleAdmin, "alice@example.com", "hashed-password")
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	users.byID[user.ID()] = user
	users.byLogin[user.Login()] = user

	hasher := &fakePasswordHasher{compareOK: true}
	tokens := &fakeTokenProvider{token: "token-1"}

	uc, err := NewLogin(users, hasher, tokens)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	result, err := uc.Execute(context.Background(), LoginCommand{
		Login:    " Alice@Example.com ",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Token != "token-1" {
		t.Fatalf("expected token token-1, got %q", result.Token)
	}
	if result.User.ID != "user-1" {
		t.Fatalf("expected user id user-1, got %q", result.User.ID)
	}
	if hasher.lastHash != "hashed-password" || hasher.lastCompare != "secret" {
		t.Fatalf("expected password compare to use stored hash and input password")
	}
	if tokens.lastUser == nil || tokens.lastUser.ID() != "user-1" {
		t.Fatalf("expected token provider to receive user")
	}
}

func TestLoginFailsWhenLoginNotFound(t *testing.T) {
	users := newFakeUserRepo()
	hasher := &fakePasswordHasher{compareOK: true}
	tokens := &fakeTokenProvider{}

	uc, err := NewLogin(users, hasher, tokens)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	_, err = uc.Execute(context.Background(), LoginCommand{
		Login:    "missing@example.com",
		Password: "secret",
	})
	if err != ErrInvalidCredentials {
		t.Fatalf("expected %v, got %v", ErrInvalidCredentials, err)
	}
}

func TestLoginFailsWhenPasswordInvalid(t *testing.T) {
	users := newFakeUserRepo()
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleAdmin, "alice@example.com", "hashed-password")
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	users.byID[user.ID()] = user
	users.byLogin[user.Login()] = user

	hasher := &fakePasswordHasher{compareOK: false}
	tokens := &fakeTokenProvider{}

	uc, err := NewLogin(users, hasher, tokens)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	_, err = uc.Execute(context.Background(), LoginCommand{
		Login:    "alice@example.com",
		Password: "wrong",
	})
	if err != ErrInvalidCredentials {
		t.Fatalf("expected %v, got %v", ErrInvalidCredentials, err)
	}
}

func TestGetCurrentUserSuccess(t *testing.T) {
	users := newFakeUserRepo()
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleAdmin, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected domain error: %v", err)
	}
	users.byID[user.ID()] = user
	users.byLogin[user.Login()] = user

	uc, err := NewGetCurrentUser(users)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	result, err := uc.Execute(context.Background(), GetCurrentUserQuery{UserID: "user-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "user-1" {
		t.Fatalf("expected user id user-1, got %q", result.ID)
	}
	if result.Login != "alice@example.com" {
		t.Fatalf("expected login alice@example.com, got %q", result.Login)
	}
}

func TestGetCurrentUserFailsWhenUserNotFound(t *testing.T) {
	users := newFakeUserRepo()

	uc, err := NewGetCurrentUser(users)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	_, err = uc.Execute(context.Background(), GetCurrentUserQuery{UserID: "missing"})
	if err != ErrUserNotFound {
		t.Fatalf("expected %v, got %v", ErrUserNotFound, err)
	}
}
