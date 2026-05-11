package app

import "errors"

var (
	ErrNilUserRepository    = errors.New("user repository is nil")
	ErrNilCompanyRepository = errors.New("company repository is nil")
	ErrNilPasswordHasher    = errors.New("password hasher is nil")
	ErrNilTokenProvider     = errors.New("token provider is nil")

	ErrCompanyNotFound            = errors.New("company not found")
	ErrUserNotFound               = errors.New("user not found")
	ErrLoginAlreadyUsed           = errors.New("login already used")
	ErrInvalidCredentials         = errors.New("invalid credentials")
	ErrPasswordRequired           = errors.New("password is required")
	ErrTermsAcceptanceRequired    = errors.New("terms acceptance is required")
	ErrTermsVersionRequired       = errors.New("terms version is required")
	ErrUserIDRequired             = errors.New("user id is required")
	ErrAdminRegistrationForbidden = errors.New("self-registration as admin is forbidden")
	ErrEmailNotVerified           = errors.New("email is not verified")
	ErrNilVerificationTokens      = errors.New("email verification token repository is nil")
	ErrNilVerificationEmailSender = errors.New("verification email sender is nil")
	ErrVerificationTokenRequired  = errors.New("verification token is required")
	ErrVerificationTokenInvalid   = errors.New("verification token is invalid")
	ErrVerificationTokenExpired   = errors.New("verification token has expired")
	ErrVerificationTokenUsed      = errors.New("verification token was already used")
	ErrVerificationCooldown       = errors.New("verification email was sent recently")
	ErrVerificationEmailSend      = errors.New("verification email could not be sent")
)
