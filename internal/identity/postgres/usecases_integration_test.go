package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type fakePasswordHasher struct {
	hashValue string
}

func (h fakePasswordHasher) Hash(password string) (string, error) {
	return h.hashValue + password, nil
}

func (h fakePasswordHasher) Compare(passwordHash, password string) (bool, error) {
	return passwordHash == h.hashValue+password, nil
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

func (g fixedIDGenerator) NewCompanyID() string {
	return g.companyID
}

func (g fixedIDGenerator) NewUserID() string {
	return g.userID
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestIdentityUseCasesWithPostgresRepositories(t *testing.T) {
	db := openIntegrationDB(t, "identity-usecases")
	companyRepo := NewCompanyRepository(db)
	userRepo := NewUserRepository(db)

	registerCompany, err := identityapp.NewRegisterCompany(
		companyRepo,
		fixedIDGenerator{companyID: "company-1"},
		fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)},
	)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	company, err := registerCompany.Execute(context.Background(), identityapp.RegisterCompanyCommand{
		Name: "Acme Fish",
		INN:  "7707083893",
		OGRN: "1027700132195",
	})
	if err != nil {
		t.Fatalf("register company error: %v", err)
	}

	registerUser, err := identityapp.NewRegisterUser(
		userRepo,
		companyRepo,
		fakePasswordHasher{hashValue: "hashed:"},
		fixedIDGenerator{userID: "user-1"},
		fixedClock{now: time.Date(2024, time.April, 1, 10, 5, 0, 0, time.UTC)},
	)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	user, err := registerUser.Execute(context.Background(), identityapp.RegisterUserCommand{
		CompanyID:     company.ID,
		Name:          "Alice",
		Role:          identity.RoleAdmin,
		Login:         " Alice@Example.com ",
		Password:      "secret",
		AcceptedTerms: true,
		TermsVersion:  "2026-04-24",
	})
	if err != nil {
		t.Fatalf("register user error: %v", err)
	}
	if user.Login != "alice@example.com" {
		t.Fatalf("expected normalized login, got %q", user.Login)
	}
	storedUser, err := userRepo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("load stored user error: %v", err)
	}
	if storedUser.TermsVersion() != "2026-04-24" {
		t.Fatalf("expected stored terms version, got %q", storedUser.TermsVersion())
	}
	if storedUser.TermsAcceptedAt().IsZero() {
		t.Fatal("expected stored accepted at to be set")
	}

	login, err := identityapp.NewLogin(
		userRepo,
		fakePasswordHasher{hashValue: "hashed:"},
		fakeTokenProvider{token: "token-1"},
	)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	loginResult, err := login.Execute(context.Background(), identityapp.LoginCommand{
		Login:    " ALICE@example.com ",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("login error: %v", err)
	}
	if loginResult.Token != "token-1" {
		t.Fatalf("expected token, got %q", loginResult.Token)
	}

	getCurrentUser, err := identityapp.NewGetCurrentUser(userRepo)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	currentUser, err := getCurrentUser.Execute(context.Background(), identityapp.GetCurrentUserQuery{
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("get current user error: %v", err)
	}
	if currentUser.ID != "user-1" || !strings.EqualFold(currentUser.Login, "alice@example.com") {
		t.Fatalf("unexpected current user: %+v", currentUser)
	}
}

func TestRegisterUserWithExistingCompanyByRequisites(t *testing.T) {
	db := openIntegrationDB(t, "identity-existing-company")
	companyRepo := NewCompanyRepository(db)
	userRepo := NewUserRepository(db)

	registerCompany, err := identityapp.NewRegisterCompany(
		companyRepo,
		fixedIDGenerator{companyID: "company-1"},
		fixedClock{now: time.Date(2024, time.April, 1, 10, 0, 0, 0, time.UTC)},
	)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	firstCompany, err := registerCompany.Execute(context.Background(), identityapp.RegisterCompanyCommand{
		Name: "Acme Fish",
		INN:  "7707083893",
		OGRN: "1027700132195",
	})
	if err != nil {
		t.Fatalf("register company error: %v", err)
	}

	secondCompany, err := registerCompany.Execute(context.Background(), identityapp.RegisterCompanyCommand{
		Name: "Another Name",
		INN:  "7707083893",
		OGRN: "1027700132195",
	})
	if err != nil {
		t.Fatalf("register existing company error: %v", err)
	}
	if secondCompany.ID != firstCompany.ID {
		t.Fatalf("expected existing company id %q, got %q", firstCompany.ID, secondCompany.ID)
	}

	registerUser, err := identityapp.NewRegisterUser(
		userRepo,
		companyRepo,
		fakePasswordHasher{hashValue: "hashed:"},
		fixedIDGenerator{userID: "user-2"},
		fixedClock{now: time.Date(2024, time.April, 1, 10, 10, 0, 0, time.UTC)},
	)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	user, err := registerUser.Execute(context.Background(), identityapp.RegisterUserCommand{
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
		t.Fatalf("register user error: %v", err)
	}
	if user.CompanyID != firstCompany.ID {
		t.Fatalf("expected company id %q, got %q", firstCompany.ID, user.CompanyID)
	}
}
