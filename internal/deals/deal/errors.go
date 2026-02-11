package deal

import "errors"

var (
	// Ошибки валидации
	ErrDealIDRequired        = errors.New("deal: ID is required")
	ErrCustomerIDRequired    = errors.New("deal: customer ID is required")
	ErrSupplierIDRequired    = errors.New("deal: supplier ID is required")
	ErrSellerCompanyRequired = errors.New("deal: seller company ID is required")
	ErrWinnerCompanyRequired = errors.New("deal: winner company ID is required")
	ErrAuctionIDRequired     = errors.New("deal: auction ID is required for auction deals")
	ErrQuantityPositive      = errors.New("deal: quantity must be positive")
	ErrUnitPricePositive     = errors.New("deal: unit price must be positive")
	ErrPriceMustBePositive   = errors.New("deal: price must be positive")
	ErrFinalPricePositive    = errors.New("deal: final price must be positive")
	ErrProductNameRequired   = errors.New("deal: product name is required")
	ErrCreatedAtRequired     = errors.New("deal: created at time is required")
	ErrSnapshotMissingFields = errors.New("deal: product snapshot missing required fields")

	// Ошибки состояния
	ErrCannotConfirmDeal        = errors.New("deal: cannot confirm deal in current status")
	ErrCannotPrepareContract    = errors.New("deal: cannot prepare contract in current status")
	ErrContractAlreadyPrepared  = errors.New("deal: contract already prepared")
	ErrCannotSignContract       = errors.New("deal: cannot sign contract in current status")
	ErrContractAlreadySigned    = errors.New("deal: contract already signed")
	ErrContractNumberRequired   = errors.New("deal: contract number is required")
	ErrCannotRequestPayment     = errors.New("deal: cannot request payment in current status")
	ErrPaymentAlreadyRequested  = errors.New("deal: payment already requested")
	ErrCannotMarkAsPaid         = errors.New("deal: cannot mark as paid in current status")
	ErrCannotRequestShipment    = errors.New("deal: cannot request shipment in current status")
	ErrShipmentAlreadyRequested = errors.New("deal: shipment already requested")
	ErrCannotMarkAsShipped      = errors.New("deal: cannot mark as shipped in current status")
	ErrCannotCompleteDeal       = errors.New("deal: cannot complete deal in current status")
	ErrCannotCancelDeal         = errors.New("deal: cannot cancel deal in current status")
	ErrCannotUpdatePrice        = errors.New("deal: cannot update price in current status")

	// Ошибки контракта
	ErrContractNotPrepared = errors.New("deal: contract not prepared")
	ErrContractNotSigned   = errors.New("deal: contract not signed")

	ErrProjectionRequired  = errors.New("deal: projection is required")
	ErrProjectionNotActive = errors.New("deal: projection is not active")
	ErrProjectionNotFound  = errors.New("deal: projection not found")

	// Удаленные ошибки
	// ErrOnlyDraftCanBeCompleted    = errors.New("deal: only draft deals can be completed from auction")
	// ErrDraftDealRequired          = errors.New("deal: draft deal is required")
)
