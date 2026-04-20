package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type CompanyRepository struct {
	db *sql.DB
}

func NewCompanyRepository(db *sql.DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

var _ app.CompanyRepository = (*CompanyRepository)(nil)

func (r *CompanyRepository) Save(ctx context.Context, company *identity.Company) error {
	const query = `
INSERT INTO identity_companies (
    company_id,
    name,
    inn,
    ogrn,
    status,
    created_at
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (company_id) DO UPDATE SET
    name = EXCLUDED.name,
    inn = EXCLUDED.inn,
    ogrn = EXCLUDED.ogrn,
    status = EXCLUDED.status,
    created_at = EXCLUDED.created_at
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(
		ctx,
		query,
		company.ID(),
		company.Name(),
		company.INN(),
		company.OGRN(),
		string(company.Status()),
		company.CreatedAt(),
	)
	return err
}

func (r *CompanyRepository) GetByID(ctx context.Context, companyID string) (*identity.Company, error) {
	const query = `
SELECT company_id, name, inn, ogrn, status, created_at
FROM identity_companies
WHERE company_id = $1
`
	return r.getOne(ctx, query, companyID)
}

func (r *CompanyRepository) GetByRequisites(ctx context.Context, inn string, ogrn string) (*identity.Company, error) {
	const query = `
SELECT company_id, name, inn, ogrn, status, created_at
FROM identity_companies
WHERE inn = $1 AND ogrn = $2
`
	return r.getOne(ctx, query, inn, ogrn)
}

func (r *CompanyRepository) ExistsByID(ctx context.Context, companyID string) (bool, error) {
	const query = `SELECT 1 FROM identity_companies WHERE company_id = $1`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, companyID)
	var exists int
	if err := row.Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *CompanyRepository) getOne(ctx context.Context, query string, args ...any) (*identity.Company, error) {
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, args...)

	var (
		id        string
		name      string
		inn       string
		ogrn      string
		status    string
		createdAt sql.NullTime
	)
	if err := row.Scan(&id, &name, &inn, &ogrn, &status, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrCompanyNotFound
		}
		return nil, err
	}
	if !createdAt.Valid {
		return nil, identity.ErrEmptyCompanyCreated
	}

	company, err := identity.NewCompany(id, name, inn, ogrn, createdAt.Time)
	if err != nil {
		return nil, err
	}

	switch identity.CompanyStatus(status) {
	case identity.CompanyStatusActive:
		return company, nil
	case identity.CompanyStatusBlocked:
		if err := company.Block(); err != nil {
			return nil, err
		}
		return company, nil
	case identity.CompanyStatusArchived:
		if err := company.Archive(); err != nil {
			return nil, err
		}
		return company, nil
	default:
		return nil, fmt.Errorf("unsupported company status: %s", status)
	}
}
