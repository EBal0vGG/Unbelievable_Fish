package app

import (
	"context"
	"errors"
	"strings"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type RegisterUser struct {
	users     UserRepository
	companies CompanyRepository
	hasher    PasswordHasher
	ids       IDGenerator
	clock     Clock
}

func NewRegisterUser(
	users UserRepository,
	companies CompanyRepository,
	hasher PasswordHasher,
	ids IDGenerator,
	clock Clock,
) (*RegisterUser, error) {
	if users == nil {
		return nil, ErrNilUserRepository
	}
	if companies == nil {
		return nil, ErrNilCompanyRepository
	}
	if hasher == nil {
		return nil, ErrNilPasswordHasher
	}
	if ids == nil {
		ids = NewRandomIDGenerator()
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &RegisterUser{
		users:     users,
		companies: companies,
		hasher:    hasher,
		ids:       ids,
		clock:     clock,
	}, nil
}

func (uc *RegisterUser) Execute(ctx context.Context, cmd RegisterUserCommand) (UserDTO, error) {
	if strings.TrimSpace(cmd.Password) == "" {
		return UserDTO{}, ErrPasswordRequired
	}
	if !cmd.AcceptedTerms {
		return UserDTO{}, ErrTermsAcceptanceRequired
	}
	if strings.TrimSpace(cmd.TermsVersion) == "" {
		return UserDTO{}, ErrTermsVersionRequired
	}

	companyID, err := uc.resolveCompanyID(ctx, cmd)
	if err != nil {
		return UserDTO{}, err
	}

	login := strings.ToLower(strings.TrimSpace(cmd.Login))
	existing, err := uc.users.GetByLogin(ctx, login)
	if err == nil && existing != nil {
		return UserDTO{}, ErrLoginAlreadyUsed
	}
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return UserDTO{}, err
	}

	passwordHash, err := uc.hasher.Hash(cmd.Password)
	if err != nil {
		return UserDTO{}, err
	}

	user, err := identity.NewUser(
		uc.ids.NewUserID(),
		companyID,
		cmd.Name,
		cmd.Role,
		login,
		passwordHash,
	)
	if err != nil {
		return UserDTO{}, err
	}
	if err := user.AcceptTerms(cmd.TermsVersion, uc.clock.Now()); err != nil {
		return UserDTO{}, err
	}
	if err := uc.users.Save(ctx, user); err != nil {
		return UserDTO{}, err
	}
	return userDTOFromDomain(user), nil
}

func (uc *RegisterUser) resolveCompanyID(ctx context.Context, cmd RegisterUserCommand) (string, error) {
	companyID := strings.TrimSpace(cmd.CompanyID)
	if companyID != "" {
		if _, err := uc.companies.GetByID(ctx, companyID); err != nil {
			if errors.Is(err, ErrCompanyNotFound) {
				return "", ErrCompanyNotFound
			}
			return "", err
		}
		return companyID, nil
	}

	companyINN := strings.TrimSpace(cmd.CompanyINN)
	companyOGRN := strings.TrimSpace(cmd.CompanyOGRN)
	if companyINN == "" || companyOGRN == "" {
		return "", identity.ErrEmptyCompanyID
	}

	company, err := uc.companies.GetByRequisites(ctx, companyINN, companyOGRN)
	if err != nil {
		if errors.Is(err, ErrCompanyNotFound) {
			return "", ErrCompanyNotFound
		}
		return "", err
	}
	return company.ID(), nil
}
