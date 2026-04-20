package postgres

import (
	"context"
	"testing"
	"time"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

func TestCompanyRepositorySaveAndLoad(t *testing.T) {
	db := openIntegrationDB(t, "company-repo")
	repo := NewCompanyRepository(db)

	company, err := identity.NewCompany("company-1", "Acme Fish", "7707083893", "1027700132195", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := company.Block(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := repo.Save(context.Background(), company); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := repo.GetByID(context.Background(), company.ID())
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.ID() != company.ID() || loaded.Status() != identity.CompanyStatusBlocked {
		t.Fatalf("unexpected company loaded: %+v", loaded)
	}

	exists, err := repo.ExistsByID(context.Background(), company.ID())
	if err != nil {
		t.Fatalf("exists error: %v", err)
	}
	if !exists {
		t.Fatal("expected company to exist")
	}

	loadedByRequisites, err := repo.GetByRequisites(context.Background(), company.INN(), company.OGRN())
	if err != nil {
		t.Fatalf("load by requisites error: %v", err)
	}
	if loadedByRequisites.ID() != company.ID() {
		t.Fatalf("unexpected company by requisites: %+v", loadedByRequisites)
	}
}

func TestUserRepositorySaveAndLoad(t *testing.T) {
	db := openIntegrationDB(t, "user-repo")
	companyRepo := NewCompanyRepository(db)
	userRepo := NewUserRepository(db)

	company, err := identity.NewCompany("company-1", "Acme Fish", "7707083893", "1027700132195", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := companyRepo.Save(context.Background(), company); err != nil {
		t.Fatalf("save company error: %v", err)
	}

	user, err := identity.NewUser("user-1", company.ID(), "Alice", identity.RoleAdmin, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := userRepo.Save(context.Background(), user); err != nil {
		t.Fatalf("save user error: %v", err)
	}

	loadedByID, err := userRepo.GetByID(context.Background(), user.ID())
	if err != nil {
		t.Fatalf("load by id error: %v", err)
	}
	if loadedByID.Login() != user.Login() {
		t.Fatalf("unexpected user by id: %+v", loadedByID)
	}

	loadedByLogin, err := userRepo.GetByLogin(context.Background(), user.Login())
	if err != nil {
		t.Fatalf("load by login error: %v", err)
	}
	if loadedByLogin.ID() != user.ID() {
		t.Fatalf("unexpected user by login: %+v", loadedByLogin)
	}

	exists, err := userRepo.ExistsByLogin(context.Background(), user.Login())
	if err != nil {
		t.Fatalf("exists error: %v", err)
	}
	if !exists {
		t.Fatal("expected user to exist")
	}
}
