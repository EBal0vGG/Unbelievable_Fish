package deal

import "errors"

// многие добавлены на будущее
var (
	// Общие ошибки
	ErrDealNotFound     = errors.New("deal not found")
	ErrInvalidDealState = errors.New("invalid deal state")
	ErrInvalidDealType  = errors.New("invalid deal type")

	// Создание сделки
	ErrAuctionIDRequired       = errors.New("auction ID is required")
	ErrSellerCompanyRequired   = errors.New("seller company ID is required")
	ErrProductNameRequired     = errors.New("product name is required")
	ErrWinnerCompanyRequired   = errors.New("winner company ID is required")
	ErrFinalPricePositive      = errors.New("final price must be positive")
	ErrDraftDealRequired       = errors.New("draft deal is required")
	ErrOnlyDraftCanBeCompleted = errors.New("only drafted deals can be completed")

	// Статусные ошибки
	ErrCannotConfirmDeal = errors.New("deal cannot be confirmed in current status")
	ErrCannotCancelDeal  = errors.New("cannot cancel already completed or cancelled deal")
	ErrCannotModifyDeal  = errors.New("cannot modify deal in current status")
	ErrCannotUpdatePrice = errors.New("cannot update price in current status")

	// Валидация данных
	ErrDealIDRequired     = errors.New("deal ID is required")
	ErrCustomerIDRequired = errors.New("customer ID is required for non-draft deals")
	ErrSupplierIDRequired = errors.New("supplier ID is required")
	ErrQuantityPositive   = errors.New("quantity must be positive")
	ErrUnitPricePositive  = errors.New("unit price must be positive")
	ErrCreatedAtRequired  = errors.New("created at time is required")

	// Ошибки подтверждения
	ErrDealNotConfirmed     = errors.New("deal is not confirmed")
	ErrDealAlreadyConfirmed = errors.New("deal is already confirmed")

	// Ошибки оплаты
	ErrCannotRequestPayment    = errors.New("cannot request payment in current status")
	ErrDealNotPaid             = errors.New("deal is not paid")
	ErrCannotMarkAsPaid        = errors.New("deal cannot be marked as paid in current status")
	ErrPaymentAlreadyRequested = errors.New("payment is already requested")

	// Ошибки доставки
	ErrCannotRequestShipment    = errors.New("cannot request shipment in current status")
	ErrCannotMarkAsShipped      = errors.New("deal cannot be marked as shipped in current status")
	ErrDealNotShipped           = errors.New("deal is not shipped")
	ErrShipmentAlreadyRequested = errors.New("shipment is already requested")

	// Ошибки завершения
	ErrCannotCompleteDeal = errors.New("deal cannot be completed in current status")

	// Цены
	ErrPriceMustBePositive = errors.New("price must be positive")
	ErrInvalidPriceUpdate  = errors.New("invalid price update")

	// Проекции/снапшоты
	ErrProductSnapshotInvalid = errors.New("product snapshot is invalid")
	ErrSnapshotMissingFields  = errors.New("product snapshot missing required fields")

	// Бизнес-логика
	ErrDealAlreadyActive    = errors.New("deal is already active")
	ErrDealAlreadyCompleted = errors.New("deal is already completed")
	ErrDealAlreadyCancelled = errors.New("deal is already cancelled")
	ErrInvalidTotalAmount   = errors.New("invalid total amount")
	ErrCurrencyMismatch     = errors.New("currency mismatch")
	ErrAuctionDealRequired  = errors.New("auction deal type required")
	ErrDirectDealRequired   = errors.New("direct deal type required")
)
