package app

import (
	"context"
	"errors"
	"strings"

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
	existing, err := uc.companies.GetByRequisites(ctx, strings.TrimSpace(cmd.INN), strings.TrimSpace(cmd.OGRN))
	if err == nil && existing != nil {
		return companyDTOFromDomain(existing), nil
	}
	if err != nil && !errors.Is(err, ErrCompanyNotFound) {
		return CompanyDTO{}, err
	}

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
