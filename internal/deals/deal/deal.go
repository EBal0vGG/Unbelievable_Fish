package deal

import (
	"time"
)

// Deal - основная сущность сделки
type Deal struct {
	id          string
	customerID  string
	supplierID  string
	auctionID   string // обязателен для аукционных сделок
	quantity    int64
	unitPrice   int64 // rubles
	status      DealStatus
	typeName    DealType
	createdAt   time.Time
	confirmedAt *time.Time
	productSnapshot
}

// ProductSnapshot - снимок продукта на момент публикации лота
type productSnapshot struct {
	ProductID     string // для справки, в запросах использовать не будем
	Name          string
	Description   string
	Category      string
	Weight        float64
	Volume        float64
	OriginCountry string
}

// DealType - тип сделки
type DealType string

const (
	DealTypeDirect  DealType = "direct"  // Прямая покупка (если понадобится в будущем)
	DealTypeAuction DealType = "auction" // Аукцион
)

// DealStatus - статус сделки
type DealStatus string

const (
	DealStatusDrafted           DealStatus = "drafted"            // драфт сделки (создан при LotPublished)
	DealStatusPending           DealStatus = "pending"            // Ожидает подтверждения (после AuctionWon)
	DealStatusConfirmed         DealStatus = "confirmed"          // Подтверждена
	DealStatusPaymentRequested  DealStatus = "payment_requested"  // Запрос на оплату - событие создание контракта
	DealStatusPaid              DealStatus = "paid"               // Оплачена
	DealStatusShipmentRequested DealStatus = "shipment_requested" // Запрос на доставку - событие начать доставку
	DealStatusShipped           DealStatus = "shipped"            // Отправлена
	DealStatusCompleted         DealStatus = "completed"          // Завершена
	DealStatusCancelled         DealStatus = "cancelled"          // Отменена
)

// Конструкторы на основе событий

// NewDealFromLotPublished - создает драфт сделки при публикации лота
func NewDealFromLotPublished(auctionID, sellerCompanyID string, productSnapshot productSnapshot, startPrice int64) (*Deal, error) {
	if auctionID == "" {
		return nil, ErrAuctionIDRequired
	}
	if sellerCompanyID == "" {
		return nil, ErrSellerCompanyRequired
	}
	if productSnapshot.Name == "" {
		return nil, ErrProductNameRequired
	}
	if startPrice <= 0 {
		return nil, ErrPriceMustBePositive
	}

	return &Deal{
		id:              generateID(),
		supplierID:      sellerCompanyID,
		auctionID:       auctionID,
		quantity:        1,          // Аукционные сделки всегда за 1 единицу
		unitPrice:       startPrice, // Начальная цена
		status:          DealStatusDrafted,
		typeName:        DealTypeAuction,
		createdAt:       time.Now(),
		productSnapshot: productSnapshot,
	}, nil
}

// CompleteDealFromAuctionWon - завершает создание сделки при выигрыше аукциона
// Принимает существующую драфт-сделку и обновляет ее
func CompleteDealFromAuctionWon(draftDeal *Deal, winnerCompanyID string, finalPrice int64) (*Deal, error) {
	if draftDeal == nil {
		return nil, ErrDraftDealRequired
	}
	if draftDeal.status != DealStatusDrafted {
		return nil, ErrOnlyDraftCanBeCompleted
	}
	if winnerCompanyID == "" {
		return nil, ErrWinnerCompanyRequired
	}
	if finalPrice <= 0 {
		return nil, ErrFinalPricePositive
	}

	draftDeal.customerID = winnerCompanyID
	draftDeal.unitPrice = finalPrice
	draftDeal.status = DealStatusPending

	return draftDeal, nil
}

// Getter методы для приватных полей

// ID возвращает идентификатор сделки
func (d *Deal) ID() string {
	return d.id
}

// CustomerID возвращает идентификатор покупателя
func (d *Deal) CustomerID() string {
	return d.customerID
}

// SupplierID возвращает идентификатор продавца
func (d *Deal) SupplierID() string {
	return d.supplierID
}

// AuctionID возвращает идентификатор аукциона
func (d *Deal) AuctionID() string {
	return d.auctionID
}

// Quantity возвращает количество товара
func (d *Deal) Quantity() int64 {
	return d.quantity
}

// UnitPrice возвращает цену за единицу
func (d *Deal) UnitPrice() int64 {
	return d.unitPrice
}

// Status возвращает статус сделки
func (d *Deal) Status() DealStatus {
	return d.status
}

// Type возвращает тип сделки
func (d *Deal) Type() DealType {
	return d.typeName
}

// CreatedAt возвращает время создания
func (d *Deal) CreatedAt() time.Time {
	return d.createdAt
}

// ConfirmedAt возвращает время подтверждения
func (d *Deal) ConfirmedAt() *time.Time {
	return d.confirmedAt
}

// ProductName возвращает название продукта
func (d *Deal) ProductName() string {
	return d.productSnapshot.Name
}

// ProductDescription возвращает описание продукта
func (d *Deal) ProductDescription() string {
	return d.productSnapshot.Description
}

// ProductID возвращает идентификатор продукта
func (d *Deal) ProductID() string {
	return d.productSnapshot.ProductID
}

// ProductCategory возвращает категорию продукта
func (d *Deal) ProductCategory() string {
	return d.productSnapshot.Category
}

// ProductWeight возвращает вес продукта
func (d *Deal) ProductWeight() float64 {
	return d.productSnapshot.Weight
}

// ProductVolume возвращает объем продукта
func (d *Deal) ProductVolume() float64 {
	return d.productSnapshot.Volume
}

// ProductOriginCountry возвращает страну происхождения продукта
func (d *Deal) ProductOriginCountry() string {
	return d.productSnapshot.OriginCountry
}

// Бизнес-методы

// CalculateTotal - вычисляет общую сумму сделки
func (d *Deal) CalculateTotal() int64 {
	return d.quantity * d.unitPrice
}

// Confirm - подтверждает сделку
func (d *Deal) Confirm() error {
	if !d.CanBeConfirmed() {
		return ErrCannotConfirmDeal
	}

	now := time.Now()
	d.status = DealStatusConfirmed
	d.confirmedAt = &now
	return nil
}

// RequestPayment - запрашивает оплату сделки
func (d *Deal) RequestPayment() error {
	if d.status != DealStatusConfirmed && d.status != DealStatusPaymentRequested {
		return ErrCannotRequestPayment
	}

	if d.status == DealStatusPaymentRequested {
		return ErrPaymentAlreadyRequested
	}

	d.status = DealStatusPaymentRequested
	return nil
}

// MarkAsPaid - отмечает сделку как оплаченную
func (d *Deal) MarkAsPaid() error {
	if d.status != DealStatusPaymentRequested {
		return ErrCannotMarkAsPaid
	}

	d.status = DealStatusPaid
	return nil
}

// RequestShipment - запрашивает доставку сделки
func (d *Deal) RequestShipment() error {
	if d.status != DealStatusPaid && d.status != DealStatusShipmentRequested {
		return ErrCannotRequestShipment
	}

	if d.status == DealStatusShipmentRequested {
		return ErrShipmentAlreadyRequested
	}

	d.status = DealStatusShipmentRequested
	return nil
}

// MarkAsShipped - отмечает сделку как отправленную
func (d *Deal) MarkAsShipped() error {
	if d.status != DealStatusShipmentRequested {
		return ErrCannotMarkAsShipped
	}

	d.status = DealStatusShipped
	return nil
}

// Complete - завершает сделку
func (d *Deal) Complete() error {
	if d.status != DealStatusShipped {
		return ErrCannotCompleteDeal
	}

	d.status = DealStatusCompleted
	return nil
}

// Cancel - отменяет сделку
func (d *Deal) Cancel() error {
	if d.status == DealStatusCompleted || d.status == DealStatusCancelled {
		return ErrCannotCancelDeal
	}

	d.status = DealStatusCancelled
	return nil
}

// UpdatePrice - обновляет цену за единицу
func (d *Deal) UpdatePrice(newPrice int64) error {
	if !d.CanBeModified() {
		return ErrCannotUpdatePrice
	}
	if newPrice <= 0 {
		return ErrPriceMustBePositive
	}

	d.unitPrice = newPrice
	return nil
}

// Validate - валидирует данные сделки
func (d *Deal) Validate() error {
	if d.id == "" {
		return ErrDealIDRequired
	}
	if d.customerID == "" && d.status != DealStatusDrafted {
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

// ValidateProductSnapshot - валидирует снимок продукта
func (d *Deal) ValidateProductSnapshot() error {
	if d.productSnapshot.Name == "" {
		return ErrProductNameRequired
	}
	if d.productSnapshot.ProductID == "" {
		return ErrSnapshotMissingFields
	}
	return nil
}

// Хелпер-методы для проверки состояний

// CanBeConfirmed - можно ли подтвердить сделку
func (d *Deal) CanBeConfirmed() bool {
	return d.status == DealStatusDrafted || d.status == DealStatusPending
}

// CanBeModified - можно ли изменять сделку
func (d *Deal) CanBeModified() bool {
	return d.status == DealStatusDrafted || d.status == DealStatusPending
}

// IsActive - активна ли сделка
func (d *Deal) IsActive() bool {
	return d.status == DealStatusDrafted ||
		d.status == DealStatusPending ||
		d.status == DealStatusConfirmed ||
		d.status == DealStatusPaymentRequested ||
		d.status == DealStatusPaid ||
		d.status == DealStatusShipmentRequested ||
		d.status == DealStatusShipped
}

// IsCompleted - завершена ли сделка
func (d *Deal) IsCompleted() bool {
	return d.status == DealStatusCompleted
}

// IsCancelled - отменена ли сделка
func (d *Deal) IsCancelled() bool {
	return d.status == DealStatusCancelled
}

// IsAuctionDeal - является ли сделка аукционной
func (d *Deal) IsAuctionDeal() bool {
	return d.typeName == DealTypeAuction
}

// IsDirectDeal - является ли сделка прямой покупкой
func (d *Deal) IsDirectDeal() bool {
	return d.typeName == DealTypeDirect
}

// IsDraft - является ли сделка драфтом
func (d *Deal) IsDraft() bool {
	return d.status == DealStatusDrafted
}

// IsPending - ожидает ли сделка подтверждения
func (d *Deal) IsPending() bool {
	return d.status == DealStatusPending
}

// IsConfirmed - подтверждена ли сделка
func (d *Deal) IsConfirmed() bool {
	return d.status == DealStatusConfirmed
}

// IsPaymentRequested - запрошена ли оплата
func (d *Deal) IsPaymentRequested() bool {
	return d.status == DealStatusPaymentRequested
}

// IsPaid - оплачена ли сделка
func (d *Deal) IsPaid() bool {
	return d.status == DealStatusPaid
}

// IsShipmentRequested - запрошена ли доставка
func (d *Deal) IsShipmentRequested() bool {
	return d.status == DealStatusShipmentRequested
}

// IsShipped - отправлена ли сделка
func (d *Deal) IsShipped() bool {
	return d.status == DealStatusShipped
}

// CanRequestPayment - можно ли запросить оплату
func (d *Deal) CanRequestPayment() bool {
	return d.status == DealStatusConfirmed
}

// CanRequestShipment - можно ли запросить доставку
func (d *Deal) CanRequestShipment() bool {
	return d.status == DealStatusPaid
}

// CanBeCompleted - можно ли завершить сделку
func (d *Deal) CanBeCompleted() bool {
	return d.status == DealStatusShipped
}

// Private helpers

// generateID - генерирует ID для сделки. можно переделать под общий формат генерации id.
func generateID() string {
	return "deal_" + time.Now().Format("20060102150405")
}
