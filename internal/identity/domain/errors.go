package identity

import "errors"

var (
	ErrEmptyUserID          = errors.New("user id is required")
	ErrEmptyCompanyID       = errors.New("company id is required")
	ErrEmptyUserName        = errors.New("user name is required")
	ErrEmptyLogin           = errors.New("login is required")
	ErrInvalidLogin         = errors.New("login must be a valid email address")
	ErrInvalidRole          = errors.New("role is invalid")
	ErrEmptyPasswordHash    = errors.New("password hash is required")
	ErrEmptyTermsVersion    = errors.New("terms version is required")
	ErrEmptyTermsAcceptedAt = errors.New("terms accepted at is required")

	ErrEmptyCompanyName    = errors.New("company name is required")
	ErrCompanyNameTooLong  = errors.New("company name is too long")
	ErrInvalidINN          = errors.New("inn is invalid")
	ErrInvalidOGRN         = errors.New("ogrn is invalid")
	ErrEmptyCompanyCreated = errors.New("company created at is required")
	ErrCompanyRenameDenied = errors.New("company rename is not allowed")
	ErrInvalidCompanyState = errors.New("company status transition is not allowed")
)
