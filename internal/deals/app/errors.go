package app

import "errors"

var (
	ErrDealRepositoryRequired       = errors.New("app: deal repository is required")
	ErrProjectionRepositoryRequired = errors.New("app: projection repository is required")
	ErrEventPublisherRequired       = errors.New("app: event publisher is required")

	ErrDealNotFound = errors.New("app: deal not found")

	ErrAuctionIDRequired     = errors.New("app: auction ID is required")
	ErrDealIDRequired        = errors.New("app: deal ID is required")
	ErrWinnerCompanyRequired = errors.New("app: winner company ID is required")
	ErrFinalPriceRequired    = errors.New("app: final price must be positive")
	ErrPublishedAtRequired   = errors.New("app: published at is required")
	ErrWonAtRequired         = errors.New("app: won at is required")

	ErrSupplierIDRequired       = errors.New("app: supplier ID is required")
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
