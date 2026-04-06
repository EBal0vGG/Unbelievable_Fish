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

// CreateFromSelection creates a deal using winner-selection data without projection state changes.
func (f *Factory) CreateFromSelection(
	auctionID string,
	supplierID string,
	snapshot ProductSnapshot,
	winnerCompanyID string,
	finalPrice int64,
	wonAt time.Time,
) (*Deal, []Event, error) {
	if auctionID == "" {
		return nil, nil, ErrAuctionIDRequired
	}
	if supplierID == "" {
		return nil, nil, ErrSupplierIDRequired
	}
	if winnerCompanyID == "" {
		return nil, nil, ErrWinnerCompanyRequired
	}
	if finalPrice <= 0 {
		return nil, nil, ErrPriceMustBePositive
	}
	if wonAt.IsZero() {
		return nil, nil, ErrCreatedAtRequired
	}

	item := &Deal{
		id:              generateID(),
		customerID:      winnerCompanyID,
		supplierID:      supplierID,
		auctionID:       auctionID,
		quantity:        1,
		unitPrice:       finalPrice,
		status:          DealStatusPending,
		typeName:        DealTypeAuction,
		createdAt:       wonAt,
		productSnapshot: snapshot,
	}

	events := []Event{
		DealCreated{
			DealID:          item.id,
			AuctionID:       auctionID,
			CustomerID:      winnerCompanyID,
			SupplierID:      supplierID,
			ProductSnapshot: snapshot,
			FinalPrice:      finalPrice,
			CreatedAt:       wonAt,
		},
	}

	return item, events, nil
}
