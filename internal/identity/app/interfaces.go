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
	List(ctx context.Context) ([]*identity.User, error)
}

type EmailVerificationToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	SentAt    time.Time
	UsedAt    *time.Time
	RevokedAt *time.Time
}

type EmailVerificationTokenRepository interface {
	Save(ctx context.Context, token EmailVerificationToken) error
	GetByHash(ctx context.Context, tokenHash string) (EmailVerificationToken, error)
	MarkUsed(ctx context.Context, tokenID string, usedAt time.Time) error
	RevokeActiveForUser(ctx context.Context, userID string, revokedAt time.Time) error
	LastSentAtForUser(ctx context.Context, userID string) (time.Time, bool, error)
}

type CompanyRepository interface {
	Save(ctx context.Context, company *identity.Company) error
	GetByID(ctx context.Context, companyID string) (*identity.Company, error)
	GetByRequisites(ctx context.Context, inn string, ogrn string) (*identity.Company, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(passwordHash, password string) (bool, error)
}

type TokenProvider interface {
	Generate(user *identity.User) (string, error)
}

type VerificationEmail struct {
	To               string
	VerificationLink string
	ExpiresAt        time.Time
}

type VerificationEmailSender interface {
	SendVerificationEmail(ctx context.Context, email VerificationEmail) error
}

type IDGenerator interface {
	NewCompanyID() string
	NewUserID() string
}

type Clock interface {
	Now() time.Time
}

// CompanyVerifier validates company requisites in an external source (e.g. FNS).
type CompanyVerifier interface {
	VerifyCompany(ctx context.Context, inn string, ogrn string) (CompanyVerificationResult, error)
}
