package httpapi

import (
	"errors"
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
)

var (
	ErrMissingCompanyID = errors.New("missing X-Company-ID header")
	ErrMissingUserID    = errors.New("missing X-User-ID header")
	ErrInvalidPath      = errors.New("invalid path")
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
	case errors.Is(err, identityauth.ErrForbidden):
		return HTTPError{http.StatusForbidden, "FORBIDDEN", "forbidden"}
	case errors.Is(err, ErrMissingCompanyID):
		return HTTPError{http.StatusBadRequest, "MISSING_COMPANY_ID", "missing X-Company-ID header"}
	case errors.Is(err, ErrMissingUserID):
		return HTTPError{http.StatusBadRequest, "MISSING_USER_ID", "missing X-User-ID header"}
	case errors.Is(err, ErrInvalidPath):
		return HTTPError{http.StatusBadRequest, "INVALID_PATH", "invalid path"}
	case errors.Is(err, app.ErrDealNotFound), errors.Is(err, deal.ErrProjectionNotFound):
		return HTTPError{http.StatusNotFound, "DEAL_NOT_FOUND", "deal not found"}
	case errors.Is(err, deal.ErrConfirmationNotFound):
		return HTTPError{http.StatusNotFound, "CONFIRMATION_NOT_FOUND", "confirmation not found"}
	case errors.Is(err, app.ErrAuctionIDRequired),
		errors.Is(err, app.ErrDealIDRequired),
		errors.Is(err, app.ErrConfirmationIDRequired),
		errors.Is(err, app.ErrWinnerCompanyRequired),
		errors.Is(err, app.ErrWinnerCandidatesRequired),
		errors.Is(err, app.ErrFinalPriceRequired),
		errors.Is(err, app.ErrPublishedAtRequired),
		errors.Is(err, app.ErrWonAtRequired),
		errors.Is(err, app.ErrSupplierIDRequired),
		errors.Is(err, app.ErrInvoiceNumberRequired),
		errors.Is(err, app.ErrPaymentIDRequired),
		errors.Is(err, app.ErrPaymentTypeRequired),
		errors.Is(err, app.ErrTrackingNumberRequired),
		errors.Is(err, app.ErrCarrierRequired),
		errors.Is(err, app.ErrReasonRequired),
		errors.Is(err, app.ErrCancelledByRequired),
		errors.Is(err, app.ErrUpdatedByRequired),
		errors.Is(err, app.ErrContractNumberRequired),
		errors.Is(err, app.ErrSignedByRequired),
		errors.Is(err, app.ErrSignatureRefRequired),
		errors.Is(err, app.ErrStartPriceMustBePositive),
		errors.Is(err, deal.ErrConfirmationIDRequired),
		errors.Is(err, deal.ErrConfirmationDealIDRequired),
		errors.Is(err, deal.ErrConfirmationStageRequired),
		errors.Is(err, deal.ErrConfirmationStatusRequired),
		errors.Is(err, deal.ErrVerificationMethodRequired),
		errors.Is(err, deal.ErrRequestedAtRequired),
		errors.Is(err, deal.ErrRequestedByUserRequired),
		errors.Is(err, deal.ErrRequestedByCompanyRequired),
		errors.Is(err, deal.ErrCounterpartyCompanyRequired),
		errors.Is(err, deal.ErrPriceMustBePositive),
		errors.Is(err, deal.ErrDealIDRequired),
		errors.Is(err, deal.ErrCustomerIDRequired),
		errors.Is(err, deal.ErrSupplierIDRequired),
		errors.Is(err, deal.ErrAuctionIDRequired),
		errors.Is(err, deal.ErrQuantityPositive),
		errors.Is(err, deal.ErrUnitPricePositive),
		errors.Is(err, deal.ErrProductNameRequired),
		errors.Is(err, deal.ErrCreatedAtRequired):
		return HTTPError{http.StatusBadRequest, "INVALID_BODY", "invalid request body"}
	case errors.Is(err, deal.ErrCounterpartyRequired):
		return HTTPError{http.StatusForbidden, "COUNTERPARTY_REQUIRED", "counterparty approval is required"}
	case errors.Is(err, deal.ErrNotDealParticipant):
		return HTTPError{http.StatusForbidden, "NOT_DEAL_PARTICIPANT", "company is not a deal participant"}
	case errors.Is(err, deal.ErrConfirmationAlreadyPending):
		return HTTPError{http.StatusConflict, "CONFIRMATION_ALREADY_PENDING", "confirmation already pending"}
	case errors.Is(err, deal.ErrCannotConfirmDeal),
		errors.Is(err, deal.ErrCannotPrepareContract),
		errors.Is(err, deal.ErrContractAlreadyPrepared),
		errors.Is(err, deal.ErrCannotSignContract),
		errors.Is(err, deal.ErrContractAlreadySigned),
		errors.Is(err, deal.ErrCannotRequestPayment),
		errors.Is(err, deal.ErrPaymentAlreadyRequested),
		errors.Is(err, deal.ErrCannotMarkAsPaid),
		errors.Is(err, deal.ErrCannotRequestShipment),
		errors.Is(err, deal.ErrShipmentAlreadyRequested),
		errors.Is(err, deal.ErrCannotMarkAsShipped),
		errors.Is(err, deal.ErrCannotCompleteDeal),
		errors.Is(err, deal.ErrCannotCancelDeal),
		errors.Is(err, deal.ErrCannotUpdatePrice),
		errors.Is(err, deal.ErrConfirmationNotPending),
		errors.Is(err, deal.ErrConfirmationExpired),
		errors.Is(err, deal.ErrConfirmationNotApproved),
		errors.Is(err, deal.ErrConfirmationDealMismatch),
		errors.Is(err, deal.ErrInvalidStageTransition),
		errors.Is(err, deal.ErrContractNotPrepared),
		errors.Is(err, deal.ErrContractNotSigned),
		errors.Is(err, deal.ErrProjectionRequired),
		errors.Is(err, deal.ErrProjectionNotActive):
		return HTTPError{http.StatusConflict, "INVALID_STAGE_TRANSITION", "invalid stage transition"}
	case errors.Is(err, app.ErrNoAvailableWinner):
		return HTTPError{http.StatusConflict, "NO_AVAILABLE_WINNER", "no available winner candidates"}
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
