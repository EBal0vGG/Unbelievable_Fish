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
	verifier  CompanyVerifier
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
		verifier:  NewNoopCompanyVerifier(),
	}, nil
}

func (uc *RegisterCompany) WithCompanyVerifier(verifier CompanyVerifier) *RegisterCompany {
	if verifier == nil {
		uc.verifier = NewNoopCompanyVerifier()
		return uc
	}
	uc.verifier = verifier
	return uc
}

func (uc *RegisterCompany) Execute(ctx context.Context, cmd RegisterCompanyCommand) (CompanyDTO, error) {
	inn := strings.TrimSpace(cmd.INN)
	ogrn := strings.TrimSpace(cmd.OGRN)
	existing, err := uc.companies.GetByRequisites(ctx, inn, ogrn)
	if err == nil && existing != nil {
		return companyDTOFromDomain(existing), nil
	}
	if err != nil && !errors.Is(err, ErrCompanyNotFound) {
		return CompanyDTO{}, err
	}
	verified, err := uc.verifier.VerifyCompany(ctx, inn, ogrn)
	if err != nil {
		return CompanyDTO{}, err
	}

	name := cmd.Name
	if strings.TrimSpace(name) == "" && strings.TrimSpace(verified.Name) != "" {
		name = verified.Name
	}

	company, err := identity.NewCompany(
		uc.ids.NewCompanyID(),
		name,
		inn,
		ogrn,
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
