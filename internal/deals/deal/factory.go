package deal

import (
	"time"
)

// Factory - создает сделки из проекций
type Factory struct{}

// NewFactory создает новую фабрику
func NewFactory() *Factory {
	return &Factory{}
}

// CreateFromProjection создает сделку из проекции и события победы
func (f *Factory) CreateFromProjection(
	projection *DealProjection,
	winnerCompanyID string,
	finalPrice int64,
	wonAt time.Time,
) (*Deal, []Event, error) {

	// Валидация
	if projection == nil {
		return nil, nil, ErrProjectionRequired
	}

	if !projection.CanBeConverted() {
		return nil, nil, ErrProjectionNotActive
	}

	if winnerCompanyID == "" {
		return nil, nil, ErrWinnerCompanyRequired
	}

	if finalPrice <= 0 {
		return nil, nil, ErrPriceMustBePositive
	}

	// Создаем сделку
	deal := &Deal{
		id:              generateID(),
		customerID:      winnerCompanyID,
		supplierID:      projection.SupplierID,
		auctionID:       projection.AuctionID,
		quantity:        1,
		unitPrice:       finalPrice,
		status:          DealStatusPending,
		typeName:        DealTypeAuction,
		createdAt:       wonAt, // время победы = время создания сделки
		productSnapshot: projection.ProductSnapshot,
	}

	events := []Event{
		DealCreated{
			DealID:          deal.id,
			AuctionID:       projection.AuctionID,
			CustomerID:      winnerCompanyID,
			SupplierID:      projection.SupplierID,
			ProductSnapshot: projection.ProductSnapshot,
			FinalPrice:      finalPrice,
			CreatedAt:       wonAt,
		},
	}

	// Отмечаем проекцию как превращенную в сделку
	projection.MarkAsConverted()

	return deal, events, nil
}
