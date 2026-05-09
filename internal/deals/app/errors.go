package app

import "errors"

var (
	ErrDealNotFound  = errors.New("app: deal not found")
	ErrNilUnitOfWork = errors.New("app: unit of work is required")

	ErrAuctionIDRequired        = errors.New("app: auction ID is required")
	ErrDealIDRequired           = errors.New("app: deal ID is required")
	ErrWinnerCompanyRequired    = errors.New("app: winner company ID is required")
	ErrFinalPriceRequired       = errors.New("app: final price must be positive")
	ErrPublishedAtRequired      = errors.New("app: published at is required")
	ErrWonAtRequired            = errors.New("app: won at is required")
	ErrWinnerCandidatesRequired = errors.New("app: winner candidates are required")
	ErrNoAvailableWinner        = errors.New("app: no available winner candidates")
	ErrMultipleActiveDealsForAuction = errors.New("app: multiple non-cancelled deals for same auction")

	ErrSupplierIDRequired       = errors.New("app: supplier ID is required")
	ErrConfirmationIDRequired   = errors.New("app: confirmation ID is required")
	ErrInvoiceNumberRequired    = errors.New("app: invoice number is required")
	ErrPaymentIDRequired        = errors.New("app: payment ID is required")
	ErrPaymentTypeRequired      = errors.New("app: payment type is required")
	ErrTrackingNumberRequired   = errors.New("app: tracking number is required")
	ErrCarrierRequired          = errors.New("app: carrier is required")
	ErrReasonRequired           = errors.New("app: reason is required")
	ErrCancelledByRequired      = errors.New("app: cancelled by is required")
	ErrUpdatedByRequired        = errors.New("app: updated by is required")
	ErrContractNumberRequired   = errors.New("app: contract number is required")
	ErrSignedByRequired         = errors.New("app: signed by is required")
	ErrSignatureRefRequired     = errors.New("app: signature ref is required")
	ErrStartPriceMustBePositive = errors.New("app: start price must be positive")
)
