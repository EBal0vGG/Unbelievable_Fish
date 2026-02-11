package deal

import (
	"time"
)

// DealProjection - проекция будущей сделки
// Создается из события LotPublished, хранит данные до победы в аукционе
type DealProjection struct {
	AuctionID       string
	SupplierID      string
	ProductSnapshot ProductSnapshot
	StartPrice      int64
	PublishedAt     time.Time
	Status          ProjectionStatus
}

// ProjectionStatus - статус проекции
type ProjectionStatus string

const (
	ProjectionStatusActive    ProjectionStatus = "active"
	ProjectionStatusConverted ProjectionStatus = "converted" // превращена в сделку
	ProjectionStatusCancelled ProjectionStatus = "cancelled" // аукцион отменен
)

// NewDealProjection создает проекцию из события публикации лота
func NewDealProjection(
	auctionID string,
	supplierID string,
	snapshot ProductSnapshot,
	startPrice int64,
	publishedAt time.Time,
) *DealProjection {
	return &DealProjection{
		AuctionID:       auctionID,
		SupplierID:      supplierID,
		ProductSnapshot: snapshot,
		StartPrice:      startPrice,
		PublishedAt:     publishedAt,
		Status:          ProjectionStatusActive,
	}
}

// CanBeConverted проверяет, можно ли превратить проекцию в сделку
func (p *DealProjection) CanBeConverted() bool {
	return p.Status == ProjectionStatusActive
}

// MarkAsConverted отмечает проекцию как превращенную в сделку
func (p *DealProjection) MarkAsConverted() {
	p.Status = ProjectionStatusConverted
}

// MarkAsCancelled отмечает проекцию как отмененную
func (p *DealProjection) MarkAsCancelled() {
	p.Status = ProjectionStatusCancelled
}
