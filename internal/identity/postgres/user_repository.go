package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

var _ app.UserRepository = (*UserRepository)(nil)

func (r *UserRepository) Save(ctx context.Context, user *identity.User) error {
	const query = `
INSERT INTO identity_users (
    user_id,
    company_id,
    name,
    role,
    login,
    password_hash,
    email_verified,
    terms_accepted_at,
    terms_version
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id) DO UPDATE SET
    company_id = EXCLUDED.company_id,
    name = EXCLUDED.name,
    role = EXCLUDED.role,
    login = EXCLUDED.login,
    password_hash = EXCLUDED.password_hash,
    email_verified = EXCLUDED.email_verified,
    terms_accepted_at = EXCLUDED.terms_accepted_at,
    terms_version = EXCLUDED.terms_version
`
	dbtx := DBTXFromContext(ctx, r.db)
	var companyID any
	if user.CompanyID() != "" {
		companyID = user.CompanyID()
	}
	var termsAcceptedAt any
	if !user.TermsAcceptedAt().IsZero() {
		termsAcceptedAt = user.TermsAcceptedAt()
	}
	var termsVersion any
	if user.TermsVersion() != "" {
		termsVersion = user.TermsVersion()
	}
	_, err := dbtx.ExecContext(
		ctx,
		query,
		user.ID(),
		companyID,
		user.Name(),
		string(user.Role()),
		user.Login(),
		user.PasswordHash(),
		user.EmailVerified(),
		termsAcceptedAt,
		termsVersion,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, userID string) (*identity.User, error) {
	const query = `
SELECT user_id, company_id, name, role, login, password_hash, email_verified, terms_accepted_at, terms_version
FROM identity_users
WHERE user_id = $1
`
	return r.getOne(ctx, query, userID)
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*identity.User, error) {
	const query = `
SELECT user_id, company_id, name, role, login, password_hash, email_verified, terms_accepted_at, terms_version
FROM identity_users
WHERE login = $1
`
	return r.getOne(ctx, query, login)
}

func (r *UserRepository) ExistsByLogin(ctx context.Context, login string) (bool, error) {
	const query = `SELECT 1 FROM identity_users WHERE login = $1`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, login)
	var exists int
	if err := row.Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *UserRepository) List(ctx context.Context) ([]*identity.User, error) {
	const query = `
SELECT user_id, company_id, name, role, login, password_hash, email_verified, terms_accepted_at, terms_version
FROM identity_users
ORDER BY user_id
`
	dbtx := DBTXFromContext(ctx, r.db)
	rows, err := dbtx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*identity.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) getOne(ctx context.Context, query string, arg string) (*identity.User, error) {
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, arg)
	return scanUser(row)
}

func scanUser(scanner interface {
	Scan(dest ...any) error
}) (*identity.User, error) {
	var (
		id              string
		companyID       sql.NullString
		name            string
		role            string
		login           string
		passwordHash    string
		emailVerified   bool
		termsAcceptedAt sql.NullTime
		termsVersion    sql.NullString
	)
	if err := scanner.Scan(&id, &companyID, &name, &role, &login, &passwordHash, &emailVerified, &termsAcceptedAt, &termsVersion); err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrUserNotFound
		}
		return nil, err
	}
	user, err := identity.NewUser(id, companyID.String, name, identity.Role(role), login, passwordHash)
	if err != nil {
		return nil, err
	}
	if emailVerified {
		user.VerifyEmail()
	}
	if termsAcceptedAt.Valid != termsVersion.Valid {
		if !termsAcceptedAt.Valid {
			return nil, identity.ErrEmptyTermsAcceptedAt
		}
		return nil, identity.ErrEmptyTermsVersion
	}
	if termsAcceptedAt.Valid {
		if err := user.AcceptTerms(termsVersion.String, termsAcceptedAt.Time); err != nil {
			return nil, err
		}
	}
	return user, nil
}
