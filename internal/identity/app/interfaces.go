package app

import (
	"context"
	"time"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type UserRepository interface {
	Save(ctx context.Context, user *identity.User) error
	GetByID(ctx context.Context, userID string) (*identity.User, error)
	GetByLogin(ctx context.Context, login string) (*identity.User, error)
}

type CompanyRepository interface {
	Save(ctx context.Context, company *identity.Company) error
	GetByID(ctx context.Context, companyID string) (*identity.Company, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(passwordHash, password string) (bool, error)
}

type TokenProvider interface {
	Generate(user *identity.User) (string, error)
}

type IDGenerator interface {
	NewCompanyID() string
	NewUserID() string
}

type Clock interface {
	Now() time.Time
}
