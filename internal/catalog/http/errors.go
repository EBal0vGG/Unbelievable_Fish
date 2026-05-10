package httpapi

import (
	"errors"
	"net/http"

	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

// ErrInvalidJSONBody is returned when the request body cannot be decoded as JSON.
var ErrInvalidJSONBody = errors.New("invalid request body")

type HTTPError struct {
	Status  int
	Code    string
	Message string
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func MapError(err error) HTTPError {
	switch {
	case errors.Is(err, identityauth.ErrMissingAuthorizationHeader):
		return HTTPError{http.StatusUnauthorized, "MISSING_AUTHORIZATION", "missing Authorization header"}
	case errors.Is(err, identityauth.ErrInvalidAuthorizationHeader):
		return HTTPError{http.StatusUnauthorized, "INVALID_AUTHORIZATION", "invalid Authorization header"}
	case errors.Is(err, identityauth.ErrInvalidToken), errors.Is(err, identityauth.ErrExpiredToken):
		return HTTPError{http.StatusUnauthorized, "INVALID_TOKEN", "invalid token"}
	case errors.Is(err, identityauth.ErrForbidden):
		return HTTPError{http.StatusForbidden, "FORBIDDEN", "forbidden"}
	case errors.Is(err, catalogapp.ErrMissingCompanyID):
		return HTTPError{http.StatusBadRequest, "MISSING_COMPANY_ID", "missing company id"}
	case errors.Is(err, catalogapp.ErrForbiddenOwner):
		return HTTPError{http.StatusForbidden, "FORBIDDEN_OWNER", "company does not own this resource"}
	case errors.Is(err, catalogapp.ErrCatalogListForbidden):
		return HTTPError{http.StatusForbidden, "FORBIDDEN", "cannot list catalog for this role"}
	case errors.Is(err, catalog.ErrInvalidIdentifier):
		return HTTPError{http.StatusBadRequest, "INVALID_PATH", "invalid path"}
	case errors.Is(err, catalogapp.ErrNotFound),
		errors.Is(err, catalogapp.ErrFishNotFound),
		errors.Is(err, catalogapp.ErrUnitNotFound),
		errors.Is(err, catalogapp.ErrProcessingTypeNotFound):
		return HTTPError{http.StatusNotFound, "NOT_FOUND", "not found"}
	case errors.Is(err, catalog.ErrForbiddenStateTransition),
		errors.Is(err, catalog.ErrPublishingRuleViolation),
		errors.Is(err, catalog.ErrModificationNotAllowed),
		errors.Is(err, catalog.ErrAlreadyAssigned):
		return HTTPError{http.StatusConflict, "CONFLICT", err.Error()}
	case errors.Is(err, catalog.ErrInvalidEnum),
		errors.Is(err, catalog.ErrInvalidQuantity),
		errors.Is(err, catalog.ErrInvalidPrice),
		errors.Is(err, catalog.ErrInvalidSchedule),
		errors.Is(err, catalog.ErrInvalidWeight),
		errors.Is(err, catalog.ErrInvalidUnit),
		errors.Is(err, catalog.ErrAuctionIDRequired):
		return HTTPError{http.StatusBadRequest, "BAD_REQUEST", err.Error()}
	case errors.Is(err, ErrInvalidJSONBody):
		return HTTPError{http.StatusBadRequest, "INVALID_BODY", "invalid request body"}
	default:
		return HTTPError{http.StatusBadRequest, "BAD_REQUEST", err.Error()}
	}
}
