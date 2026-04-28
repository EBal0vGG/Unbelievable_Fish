package identity

import (
	"strings"
	"time"
)

type User struct {
	id              string
	companyID       string
	name            string
	role            Role
	login           string
	passwordHash    string
	termsAcceptedAt time.Time
	termsVersion    string
}

func NewUser(userID string, companyID string, name string, role Role, login string, passwordHash string) (*User, error) {
	userID = strings.TrimSpace(userID)
	companyID = strings.TrimSpace(companyID)
	name = strings.TrimSpace(name)
	login = normalizeLogin(login)
	passwordHash = strings.TrimSpace(passwordHash)
	role = normalizeRole(role)

	if isBlank(userID) {
		return nil, ErrEmptyUserID
	}
	if isBlank(name) {
		return nil, ErrEmptyUserName
	}
	if isBlank(login) {
		return nil, ErrEmptyLogin
	}
	if !isValidEmail(login) {
		return nil, ErrInvalidLogin
	}
	if !IsValidRole(role) {
		return nil, ErrInvalidRole
	}
	if isBlank(passwordHash) {
		return nil, ErrEmptyPasswordHash
	}

	return &User{
		id:           userID,
		companyID:    companyID,
		name:         name,
		role:         role,
		login:        login,
		passwordHash: passwordHash,
	}, nil
}

func (u *User) ID() string {
	return u.id
}

func (u *User) CompanyID() string {
	return u.companyID
}

func (u *User) Name() string {
	return u.name
}

func (u *User) Role() Role {
	return u.role
}

func (u *User) Login() string {
	return u.login
}

func (u *User) PasswordHash() string {
	return u.passwordHash
}

func (u *User) TermsAcceptedAt() time.Time {
	return u.termsAcceptedAt
}

func (u *User) TermsVersion() string {
	return u.termsVersion
}

func (u *User) AcceptTerms(version string, acceptedAt time.Time) error {
	version = strings.TrimSpace(version)
	if isBlank(version) {
		return ErrEmptyTermsVersion
	}
	if acceptedAt.IsZero() {
		return ErrEmptyTermsAcceptedAt
	}

	u.termsVersion = version
	u.termsAcceptedAt = acceptedAt
	return nil
}

func (u *User) PromoteToAdmin() {
	u.role = RoleAdmin
}
