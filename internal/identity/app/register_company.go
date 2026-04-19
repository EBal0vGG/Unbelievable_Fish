package app

import (
	"context"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type RegisterCompany struct {
	companies CompanyRepository
	ids       IDGenerator
	clock     Clock
}

func NewRegisterCompany(companies CompanyRepository, ids IDGenerator, clock Clock) (*RegisterCompany, error) {
	if companies == nil {
		return nil, ErrNilCompanyRepository
	}
	if ids == nil {
		ids = NewRandomIDGenerator()
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &RegisterCompany{
		companies: companies,
		ids:       ids,
		clock:     clock,
	}, nil
}

func (uc *RegisterCompany) Execute(ctx context.Context, cmd RegisterCompanyCommand) (CompanyDTO, error) {
	company, err := identity.NewCompany(
		uc.ids.NewCompanyID(),
		cmd.Name,
		cmd.INN,
		cmd.OGRN,
		uc.clock.Now(),
	)
	if err != nil {
		return CompanyDTO{}, err
	}
	if err := uc.companies.Save(ctx, company); err != nil {
		return CompanyDTO{}, err
	}
	return companyDTOFromDomain(company), nil
}
