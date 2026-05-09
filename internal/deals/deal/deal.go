package deal

import (
	"time"
)

// ProductSnapshot - снимок продукта на момент создания сделки
type ProductSnapshot struct {
	ProductID      string
	Name           string
	Description    string
	Category       string
	Weight         float64
	Unit           string
	Size           string
	ProcessingType string
	Volume         float64
	OriginCountry  string
}

// ContractInfo - информация о контракте сделки
type ContractInfo struct {
	Number       string
	PreparedAt   *time.Time
	SignedAt     *time.Time
	SignedBy     string
	SignatureRef string
	DocumentURL  string
}

// Deal - основная сущность сделки
// Создается только при выигрыше аукциона
type Deal struct {
	id                   string
	customerID           string
	supplierID           string
	auctionID            string // обязателен для аукционных сделок
	quantity             int64
	unitPrice            int64 // финальная цена
	status               DealStatus
	typeName             DealType
	createdAt            time.Time
	confirmedAt          *time.Time
	contractSignDeadline *time.Time
	paymentDeadline      *time.Time
	contract             *ContractInfo
	productSnapshot      ProductSnapshot
}

// Getter методы
func (d *Deal) ID() string {
	return d.id
}

func (d *Deal) CustomerID() string {
	return d.customerID
}

func (d *Deal) SupplierID() string {
	return d.supplierID
}

func (d *Deal) AuctionID() string {
	return d.auctionID
}

func (d *Deal) Quantity() int64 {
	return d.quantity
}

func (d *Deal) UnitPrice() int64 {
	return d.unitPrice
}

func (d *Deal) Status() DealStatus {
	return d.status
}

func (d *Deal) Type() DealType {
	return d.typeName
}

func (d *Deal) CreatedAt() time.Time {
	return d.createdAt
}

func (d *Deal) ConfirmedAt() *time.Time {
	return d.confirmedAt
}

func (d *Deal) Contract() *ContractInfo {
	return d.contract
}

func (d *Deal) ContractSignDeadline() *time.Time {
	return d.contractSignDeadline
}

func (d *Deal) PaymentDeadline() *time.Time {
	return d.paymentDeadline
}

func (d *Deal) ContractNumber() string {
	if d.contract == nil {
		return ""
	}
	return d.contract.Number
}

// Геттеры для ProductSnapshot
func (d *Deal) ProductName() string {
	return d.productSnapshot.Name
}

func (d *Deal) ProductDescription() string {
	return d.productSnapshot.Description
}

func (d *Deal) ProductID() string {
	return d.productSnapshot.ProductID
}

func (d *Deal) ProductCategory() string {
	return d.productSnapshot.Category
}

func (d *Deal) ProductWeight() float64 {
	return d.productSnapshot.Weight
}

func (d *Deal) ProductVolume() float64 {
	return d.productSnapshot.Volume
}

func (d *Deal) ProductOriginCountry() string {
	return d.productSnapshot.OriginCountry
}

func (d *Deal) ProductSnapshot() ProductSnapshot {
	return d.productSnapshot
}

func (d *Deal) IsParticipant(companyID string) bool {
	return companyID != "" && (companyID == d.customerID || companyID == d.supplierID)
}

// Бизнес-методы

// CalculateTotal - вычисляет общую сумму сделки (quantity * unitPrice).
// Для аукционных сделок quantity=1 и unitPrice=финальная цена лота — итог = цена лота.
func (d *Deal) CalculateTotal() int64 {
	return d.quantity * d.unitPrice
}

// Бизнес-методы - ВСЕ возвращают ([]Event, error)

// Confirm - подтверждает сделку
func (d *Deal) Confirm() ([]Event, error) {
	if d.status != DealStatusPending {
		return nil, ErrCannotConfirmDeal
	}

	now := time.Now()
	d.status = DealStatusConfirmed
	d.confirmedAt = &now

	events := []Event{
		DealConfirmed{
			DealID:      d.id,
			ConfirmedAt: now,
		},
	}

	return events, nil
}

// PrepareContract - подготавливает контракт для сделки
func (d *Deal) PrepareContract(contractNumber, documentURL string) ([]Event, error) {
	if d.status != DealStatusConfirmed && d.status != DealStatusPending {
		return nil, ErrCannotPrepareContract
	}

	if d.contract != nil && d.contract.PreparedAt != nil {
		return nil, ErrContractAlreadyPrepared
	}

	if contractNumber == "" {
		return nil, ErrContractNumberRequired
	}

	now := time.Now()
	if d.contract == nil {
		d.contract = &ContractInfo{}
	}

	d.contract.Number = contractNumber
	d.contract.PreparedAt = &now
	d.contract.DocumentURL = documentURL
	d.status = DealStatusContractPrepared

	events := []Event{
		ContractPrepared{
			DealID:         d.id,
			ContractNumber: contractNumber,
			PreparedAt:     now,
			DocumentURL:    documentURL,
		},
	}

	return events, nil
}

// SignContract - подписывает контракт
func (d *Deal) SignContract(signedBy, signatureRef string) ([]Event, error) {
	if d.status != DealStatusContractPrepared {
		return nil, ErrCannotSignContract
	}

	if d.contract == nil || d.contract.PreparedAt == nil {
		return nil, ErrContractNotPrepared
	}

	if d.contract.SignedAt != nil {
		return nil, ErrContractAlreadySigned
	}

	now := time.Now()
	d.contract.SignedAt = &now
	d.contract.SignedBy = signedBy
	d.contract.SignatureRef = signatureRef
	defaultPaymentDeadline := now.Add(24 * time.Hour)
	d.paymentDeadline = &defaultPaymentDeadline
	d.status = DealStatusContractSigned

	events := []Event{
		ContractSigned{
			DealID:         d.id,
			ContractNumber: d.contract.Number,
			SignedAt:       now,
			SignedBy:       signedBy,
			SignatureRef:   signatureRef,
		},
	}

	return events, nil
}

// RequestPayment - запрашивает оплату сделки
func (d *Deal) RequestPayment(invoiceNumber string, dueDate *time.Time) ([]Event, error) {
	if d.status == DealStatusPaymentRequested {
		return nil, ErrPaymentAlreadyRequested
	}

	if d.status != DealStatusContractSigned {
		return nil, ErrCannotRequestPayment
	}

	if !d.hasSignedContract() {
		return nil, ErrContractNotSigned
	}

	d.status = DealStatusPaymentRequested

	now := time.Now()
	goods := d.CalculateTotal()
	events := []Event{
		PaymentRequested{
			DealID:          d.id,
			AuctionID:       d.auctionID,
			BuyerCompanyID:  d.customerID,
			SellerCompanyID: d.supplierID,
			Currency:        "RUB",
			GoodsAmount:     goods,
			InvoiceNumber:   invoiceNumber,
			DueDate:         dueDate,
			RequestedAt:     now,
		},
	}

	return events, nil
}

// MarkAsPaid - отмечает сделку как оплаченную
func (d *Deal) MarkAsPaid(paymentID, paymentType string) ([]Event, error) {
	if d.status != DealStatusPaymentRequested {
		return nil, ErrCannotMarkAsPaid
	}

	d.status = DealStatusPaid

	events := []Event{
		DealPaid{
			DealID:      d.id,
			PaymentID:   paymentID,
			PaymentType: paymentType,
			PaidAt:      time.Now(),
		},
	}

	return events, nil
}

// RequestShipment - запрашивает доставку сделки
func (d *Deal) RequestShipment() ([]Event, error) {
	if d.status != DealStatusPaid {
		return nil, ErrCannotRequestShipment
	}

	if d.status == DealStatusShipmentRequested {
		return nil, ErrShipmentAlreadyRequested
	}

	d.status = DealStatusShipmentRequested

	events := []Event{
		ShipmentRequested{
			DealID:      d.id,
			RequestedAt: time.Now(),
		},
	}

	return events, nil
}

// MarkAsShipped - отмечает сделку как отправленную
func (d *Deal) MarkAsShipped(trackingNumber, carrier string) ([]Event, error) {
	if d.status != DealStatusShipmentRequested {
		return nil, ErrCannotMarkAsShipped
	}

	d.status = DealStatusShipped

	events := []Event{
		DealShipped{
			DealID:         d.id,
			TrackingNumber: trackingNumber,
			Carrier:        carrier,
			ShippedAt:      time.Now(),
		},
	}

	return events, nil
}

// Complete - завершает сделку
func (d *Deal) Complete() ([]Event, error) {
	if d.status != DealStatusShipped {
		return nil, ErrCannotCompleteDeal
	}

	d.status = DealStatusCompleted

	events := []Event{
		DealCompleted{
			DealID:      d.id,
			CompletedAt: time.Now(),
		},
	}

	return events, nil
}

// Cancel - отменяет сделку
func (d *Deal) Cancel(reason, cancelledBy string) ([]Event, error) {
	if d.status == DealStatusCompleted || d.status == DealStatusCancelled {
		return nil, ErrCannotCancelDeal
	}

	now := time.Now()
	cancelEv := DealCancelled{
		DealID:      d.id,
		Reason:      reason,
		CancelledBy: cancelledBy,
		CancelledAt: now,
	}

	if ReasonForfeitsWinnerDeposit(reason) {
		if d.typeName != DealTypeAuction {
			return nil, ErrWinnerForfeitOnlyAuction
		}
		if d.status != DealStatusPending {
			return nil, ErrCannotDeclineWinnerAfterConfirm
		}
		if d.auctionID == "" {
			return nil, ErrAuctionIDRequired
		}
		d.status = DealStatusCancelled
		norm := NormalizeWinnerForfeitReason(reason)
		return []Event{
			cancelEv,
			WinnerRejected{
				SelectionID: d.auctionID,
				DealID:      d.id,
				AuctionID:   d.auctionID,
				CompanyID:   d.customerID,
				RejectedAt:  now,
				Reason:      norm,
			},
		}, nil
	}

	d.status = DealStatusCancelled
	return []Event{cancelEv}, nil
}

// UpdatePrice - обновляет цену за единицу
func (d *Deal) UpdatePrice(newPrice int64, updatedBy string) ([]Event, error) {
	if !d.canBeModified() {
		return nil, ErrCannotUpdatePrice
	}
	if newPrice <= 0 {
		return nil, ErrPriceMustBePositive
	}

	oldPrice := d.unitPrice
	d.unitPrice = newPrice

	events := []Event{
		PriceUpdated{
			DealID:    d.id,
			OldPrice:  oldPrice,
			NewPrice:  newPrice,
			UpdatedBy: updatedBy,
			UpdatedAt: time.Now(),
		},
	}

	return events, nil
}

func (d *Deal) RequestConfirmation(
	stage DealConfirmationStage,
	requestedByCompanyID string,
	requestedByUserID string,
	method VerificationMethod,
	verificationTokenHash string,
	signatureRef string,
	comment string,
	expiresAt *time.Time,
) (*DealConfirmation, []Event, error) {
	if !d.IsParticipant(requestedByCompanyID) {
		return nil, nil, ErrNotDealParticipant
	}
	counterpartyCompanyID, err := d.counterpartyCompanyID(requestedByCompanyID)
	if err != nil {
		return nil, nil, err
	}
	if !d.canRequestConfirmationStage(stage) {
		return nil, nil, ErrInvalidStageTransition
	}

	confirmation, err := NewDealConfirmation(DealConfirmationParams{
		DealID:                d.id,
		Stage:                 stage,
		RequestedByUserID:     requestedByUserID,
		RequestedByCompanyID:  requestedByCompanyID,
		CounterpartyCompanyID: counterpartyCompanyID,
		VerificationMethod:    method,
		VerificationTokenHash: verificationTokenHash,
		SignatureRef:          signatureRef,
		RequestedAt:           time.Now().UTC(),
		ExpiresAt:             expiresAt,
		Comment:               comment,
	})
	if err != nil {
		return nil, nil, err
	}

	return confirmation, []Event{
		DealConfirmationRequested{
			ConfirmationID:        confirmation.ID(),
			DealID:                confirmation.DealID(),
			Stage:                 confirmation.Stage(),
			RequestedByUserID:     confirmation.RequestedByUserID(),
			RequestedByCompanyID:  confirmation.RequestedByCompanyID(),
			CounterpartyCompanyID: confirmation.CounterpartyCompanyID(),
			VerificationMethod:    confirmation.VerificationMethod(),
			RequestedAt:           confirmation.RequestedAt(),
			ExpiresAt:             confirmation.ExpiresAt(),
			Comment:               confirmation.Comment(),
		},
	}, nil
}

func (d *Deal) ApplyApprovedConfirmation(confirmation *DealConfirmation) ([]Event, error) {
	if confirmation == nil {
		return nil, ErrConfirmationRequired
	}
	if confirmation.DealID() != d.id {
		return nil, ErrConfirmationDealMismatch
	}
	if confirmation.Status() != DealConfirmationStatusApproved {
		return nil, ErrConfirmationNotApproved
	}
	if !d.canRequestConfirmationStage(confirmation.Stage()) {
		return nil, ErrInvalidStageTransition
	}

	switch confirmation.Stage() {
	case DealConfirmationStageConfirmed:
		return d.Confirm()
	case DealConfirmationStagePaid:
		return d.MarkAsPaid("", "")
	case DealConfirmationStageShipped:
		return d.MarkAsShipped("", "")
	case DealConfirmationStageCompleted:
		return d.Complete()
	case DealConfirmationStageCancelled:
		reason := confirmation.Comment()
		if reason == "" {
			reason = "cancelled by approved confirmation"
		}
		return d.Cancel(reason, confirmation.RequestedByCompanyID())
	default:
		return nil, ErrInvalidStageTransition
	}
}

// Validate - валидирует данные сделки
func (d *Deal) Validate() error {
	if d.id == "" {
		return ErrDealIDRequired
	}
	if d.customerID == "" {
		return ErrCustomerIDRequired
	}
	if d.supplierID == "" {
		return ErrSupplierIDRequired
	}
	if d.auctionID == "" && d.typeName == DealTypeAuction {
		return ErrAuctionIDRequired
	}
	if d.quantity <= 0 {
		return ErrQuantityPositive
	}
	if d.unitPrice <= 0 {
		return ErrUnitPricePositive
	}
	if d.productSnapshot.Name == "" {
		return ErrProductNameRequired
	}
	if d.createdAt.IsZero() {
		return ErrCreatedAtRequired
	}

	return nil
}

// Хелпер-методы для проверки состояний

// canBeModified - можно ли изменять сделку
func (d *Deal) canBeModified() bool {
	return d.status.IsModifiable()
}

// hasSignedContract - есть ли подписанный контракт
func (d *Deal) hasSignedContract() bool {
	return d.contract != nil && d.contract.SignedAt != nil
}

// hasContract - есть ли контракт
func (d *Deal) hasContract() bool {
	return d.contract != nil && d.contract.PreparedAt != nil
}

func (d *Deal) canRequestConfirmationStage(stage DealConfirmationStage) bool {
	switch stage {
	case DealConfirmationStageConfirmed:
		return d.status == DealStatusPending
	case DealConfirmationStagePaid:
		return d.status == DealStatusPaymentRequested
	case DealConfirmationStageShipped:
		return d.status == DealStatusShipmentRequested
	case DealConfirmationStageCompleted:
		return d.status == DealStatusShipped
	case DealConfirmationStageCancelled:
		return d.status != DealStatusCompleted && d.status != DealStatusCancelled
	default:
		return false
	}
}

func (d *Deal) counterpartyCompanyID(companyID string) (string, error) {
	switch companyID {
	case d.customerID:
		return d.supplierID, nil
	case d.supplierID:
		return d.customerID, nil
	default:
		return "", ErrNotDealParticipant
	}
}

// Private helpers

func generateID() string {
	return "deal_" + time.Now().Format("20060102150405")
}
