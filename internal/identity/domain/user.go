package identity

import "strings"

type User struct {
	id           string
	companyID    string
	name         string
	role         Role
	login        string
	passwordHash string
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
	if isBlank(companyID) {
		return nil, ErrEmptyCompanyID
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
