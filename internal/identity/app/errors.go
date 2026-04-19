package app

import "errors"

var (
	ErrNilUserRepository    = errors.New("user repository is nil")
	ErrNilCompanyRepository = errors.New("company repository is nil")
	ErrNilPasswordHasher    = errors.New("password hasher is nil")
	ErrNilTokenProvider     = errors.New("token provider is nil")

	ErrCompanyNotFound    = errors.New("company not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrLoginAlreadyUsed   = errors.New("login already used")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPasswordRequired   = errors.New("password is required")
	ErrUserIDRequired     = errors.New("user id is required")
)
