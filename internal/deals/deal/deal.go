package deal

import (
	"time"
)

// ProductSnapshot - снимок продукта на момент создания сделки
type ProductSnapshot struct {
	ProductID     string
	Name          string
	Description   string
	Category      string
	Weight        float64
	Volume        float64
	OriginCountry string
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
	id              string
	customerID      string
	supplierID      string
	auctionID       string // обязателен для аукционных сделок
	quantity        int64
	unitPrice       int64 // финальная цена
	status          DealStatus
	typeName        DealType
	createdAt       time.Time
	confirmedAt     *time.Time
	contract        *ContractInfo
	productSnapshot ProductSnapshot
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

// Бизнес-методы

// CalculateTotal - вычисляет общую сумму сделки
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
	if d.status != DealStatusConfirmed {
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
	if d.status != DealStatusContractSigned {
		return nil, ErrCannotRequestPayment
	}

	if d.status == DealStatusPaymentRequested {
		return nil, ErrPaymentAlreadyRequested
	}

	if !d.hasSignedContract() {
		return nil, ErrContractNotSigned
	}

	d.status = DealStatusPaymentRequested

	events := []Event{
		PaymentRequested{
			DealID:        d.id,
			TotalAmount:   d.CalculateTotal(),
			InvoiceNumber: invoiceNumber,
			DueDate:       dueDate,
			RequestedAt:   time.Now(),
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

	d.status = DealStatusCancelled

	events := []Event{
		DealCancelled{
			DealID:      d.id,
			Reason:      reason,
			CancelledBy: cancelledBy,
			CancelledAt: time.Now(),
		},
	}

	return events, nil
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

// Private helpers

func generateID() string {
	return "deal_" + time.Now().Format("20060102150405")
}
