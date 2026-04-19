package httpapi

import (
	"errors"
	"net/http"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

var (
	ErrMissingUserID = errors.New("missing X-User-ID header")
	ErrInvalidPath   = errors.New("invalid path")
)

type HTTPError struct {
	Status  int
	Code    string
	Message string
}

func MapError(err error) HTTPError {
	var badReq badRequestError
	switch {
	case errors.As(err, &badReq):
		return HTTPError{http.StatusBadRequest, badReq.code, badReq.message}
	case errors.Is(err, identityauth.ErrMissingAuthorizationHeader):
		return HTTPError{http.StatusUnauthorized, "MISSING_AUTHORIZATION", "missing Authorization header"}
	case errors.Is(err, identityauth.ErrInvalidAuthorizationHeader):
		return HTTPError{http.StatusUnauthorized, "INVALID_AUTHORIZATION", "invalid Authorization header"}
	case errors.Is(err, identityauth.ErrInvalidToken), errors.Is(err, identityauth.ErrExpiredToken):
		return HTTPError{http.StatusUnauthorized, "INVALID_TOKEN", "invalid token"}
	case errors.Is(err, identityauth.ErrIdentityNotFound):
		return HTTPError{http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized"}
	case errors.Is(err, identityauth.ErrForbidden):
		return HTTPError{http.StatusForbidden, "FORBIDDEN", "forbidden"}
	case errors.Is(err, ErrMissingUserID):
		return HTTPError{http.StatusBadRequest, "MISSING_USER_ID", "missing X-User-ID header"}
	case errors.Is(err, ErrInvalidPath):
		return HTTPError{http.StatusBadRequest, "INVALID_PATH", "invalid path"}
	case errors.Is(err, identityapp.ErrCompanyNotFound):
		return HTTPError{http.StatusNotFound, "COMPANY_NOT_FOUND", "company not found"}
	case errors.Is(err, identityapp.ErrUserNotFound):
		return HTTPError{http.StatusNotFound, "USER_NOT_FOUND", "user not found"}
	case errors.Is(err, identityapp.ErrLoginAlreadyUsed):
		return HTTPError{http.StatusConflict, "LOGIN_ALREADY_USED", "login already used"}
	case errors.Is(err, identityapp.ErrInvalidCredentials):
		return HTTPError{http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials"}
	case errors.Is(err, identityapp.ErrPasswordRequired):
		return HTTPError{http.StatusBadRequest, "PASSWORD_REQUIRED", err.Error()}
	case errors.Is(err, identityapp.ErrUserIDRequired):
		return HTTPError{http.StatusBadRequest, "USER_ID_REQUIRED", err.Error()}
	case errors.Is(err, identity.ErrEmptyCompanyID):
		return HTTPError{http.StatusBadRequest, "COMPANY_ID_REQUIRED", err.Error()}
	case errors.Is(err, identity.ErrEmptyCompanyName):
		return HTTPError{http.StatusBadRequest, "COMPANY_NAME_REQUIRED", err.Error()}
	case errors.Is(err, identity.ErrCompanyNameTooLong):
		return HTTPError{http.StatusBadRequest, "COMPANY_NAME_TOO_LONG", err.Error()}
	case errors.Is(err, identity.ErrInvalidINN):
		return HTTPError{http.StatusBadRequest, "INVALID_INN", err.Error()}
	case errors.Is(err, identity.ErrInvalidOGRN):
		return HTTPError{http.StatusBadRequest, "INVALID_OGRN", err.Error()}
	case errors.Is(err, identity.ErrEmptyCompanyCreated):
		return HTTPError{http.StatusBadRequest, "COMPANY_CREATED_AT_REQUIRED", err.Error()}
	case errors.Is(err, identity.ErrEmptyUserID):
		return HTTPError{http.StatusBadRequest, "USER_ID_REQUIRED", err.Error()}
	case errors.Is(err, identity.ErrEmptyUserName):
		return HTTPError{http.StatusBadRequest, "USER_NAME_REQUIRED", err.Error()}
	case errors.Is(err, identity.ErrEmptyLogin):
		return HTTPError{http.StatusBadRequest, "LOGIN_REQUIRED", err.Error()}
	case errors.Is(err, identity.ErrInvalidLogin):
		return HTTPError{http.StatusBadRequest, "INVALID_LOGIN", err.Error()}
	case errors.Is(err, identity.ErrInvalidRole):
		return HTTPError{http.StatusBadRequest, "INVALID_ROLE", err.Error()}
	case errors.Is(err, identity.ErrEmptyPasswordHash):
		return HTTPError{http.StatusBadRequest, "PASSWORD_HASH_REQUIRED", err.Error()}
	default:
		return HTTPError{http.StatusInternalServerError, "INTERNAL_ERROR", "internal error"}
	}
}

type badRequestError struct {
	code    string
	message string
}

func (e badRequestError) Error() string {
	return e.code + ": " + e.message
}

func BadRequest(code, message string) error {
	return badRequestError{code: code, message: message}
}
