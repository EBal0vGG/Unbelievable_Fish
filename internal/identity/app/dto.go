package app

import (
	"time"

	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type RegisterCompanyCommand struct {
	Name string
	INN  string
	OGRN string
}

type RegisterUserCommand struct {
	CompanyID string
	Name      string
	Role      identity.Role
	Login     string
	Password  string
}

type LoginCommand struct {
	Login    string
	Password string
}

type GetCurrentUserQuery struct {
	UserID string
}

type CompanyDTO struct {
	ID        string
	Name      string
	INN       string
	OGRN      string
	Status    identity.CompanyStatus
	CreatedAt time.Time
}

type UserDTO struct {
	ID        string
	CompanyID string
	Name      string
	Role      identity.Role
	Login     string
}

type LoginResult struct {
	Token string
	User  UserDTO
}

func companyDTOFromDomain(company *identity.Company) CompanyDTO {
	return CompanyDTO{
		ID:        company.ID(),
		Name:      company.Name(),
		INN:       company.INN(),
		OGRN:      company.OGRN(),
		Status:    company.Status(),
		CreatedAt: company.CreatedAt(),
	}
}

func userDTOFromDomain(user *identity.User) UserDTO {
	return UserDTO{
		ID:        user.ID(),
		CompanyID: user.CompanyID(),
		Name:      user.Name(),
		Role:      user.Role(),
		Login:     user.Login(),
	}
}
