package httpapi

import (
	"time"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type RegisterCompanyRequest struct {
	Name string `json:"name"`
	INN  string `json:"inn"`
	OGRN string `json:"ogrn"`
}

type RegisterUserRequest struct {
	CompanyID     string        `json:"company_id"`
	CompanyINN    string        `json:"company_inn"`
	CompanyOGRN   string        `json:"company_ogrn"`
	Name          string        `json:"name"`
	Role          identity.Role `json:"role"`
	Login         string        `json:"login"`
	Password      string        `json:"password"`
	AcceptedTerms bool          `json:"accepted_terms"`
	TermsVersion  string        `json:"terms_version"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type CompanyResponse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	INN       string                 `json:"inn"`
	OGRN      string                 `json:"ogrn"`
	Status    identity.CompanyStatus `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
}

type UserResponse struct {
	ID        string        `json:"id"`
	CompanyID string        `json:"company_id"`
	Name      string        `json:"name"`
	Role      identity.Role `json:"role"`
	Login     string        `json:"login"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type ErrorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
}

func NewCompanyResponse(company identityapp.CompanyDTO) CompanyResponse {
	return CompanyResponse{
		ID:        company.ID,
		Name:      company.Name,
		INN:       company.INN,
		OGRN:      company.OGRN,
		Status:    company.Status,
		CreatedAt: company.CreatedAt,
	}
}

func NewUserResponse(user identityapp.UserDTO) UserResponse {
	return UserResponse{
		ID:        user.ID,
		CompanyID: user.CompanyID,
		Name:      user.Name,
		Role:      user.Role,
		Login:     user.Login,
	}
}

func NewLoginResponse(result identityapp.LoginResult) LoginResponse {
	return LoginResponse{
		Token: result.Token,
		User:  NewUserResponse(result.User),
	}
}
