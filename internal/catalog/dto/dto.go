package dto

import "time"

// RequestMetadataDTO содержит транспортные метаданные запроса.
// Это не бизнес-данные домена, а контекст трассировки и безопасности.
type RequestMetadataDTO struct {
	CompanyID     string `json:"company_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
}

// EventMetadataDTO содержит метаданные event envelope для outbox/broker.
// В этой структуре нет бизнес-логики, только служебные поля события.
type EventMetadataDTO struct {
	EventID       string    `json:"event_id,omitempty"`
	EventType     string    `json:"event_type,omitempty"`
	EventVersion  string    `json:"event_version,omitempty"`
	OccurredAt    time.Time `json:"occurred_at,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Producer      string    `json:"producer,omitempty"`
}

type ProductStatusDTO string
type LotStatusDTO string

const (
	ProductStatusDraftDTO       ProductStatusDTO = "DRAFT"
	ProductStatusPublishedDTO   ProductStatusDTO = "PUBLISHED"
	ProductStatusUnpublishedDTO ProductStatusDTO = "UNPUBLISHED"

	LotStatusDraftDTO     LotStatusDTO = "DRAFT"
	LotStatusPublishedDTO LotStatusDTO = "PUBLISHED"
	LotStatusClosedDTO    LotStatusDTO = "CLOSED"
	LotStatusCancelledDTO LotStatusDTO = "CANCELLED"
)

// CreateFishDTO используется для создания записи Fish.
type CreateFishDTO struct {
	FishID      string `json:"fish_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// UpdateFishDTO используется для обновления Fish.
type UpdateFishDTO struct {
	FishID      string `json:"fish_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreateProductDTO описывает входные данные для создания Product.
type CreateProductDTO struct {
	ProductID      string  `json:"product_id"`
	FishID         string  `json:"fish_id"`
	Weight         float64 `json:"weight"`
	Unit           string  `json:"unit"`
	Size           string  `json:"size,omitempty"`
	ProcessingType string  `json:"processing_type"`
}

// UpdateProductDTO описывает входные данные для изменения Product.
type UpdateProductDTO struct {
	ProductID      string  `json:"product_id"`
	FishID         string  `json:"fish_id"`
	Weight         float64 `json:"weight"`
	Unit           string  `json:"unit"`
	Size           string  `json:"size,omitempty"`
	ProcessingType string  `json:"processing_type"`
}

// PublishProductDTO используется для публикации Product.
type PublishProductDTO struct {
	ProductID string `json:"product_id"`
}

// UnpublishProductDTO используется для снятия Product с публикации.
type UnpublishProductDTO struct {
	ProductID string `json:"product_id"`
}

// CreateLotDTO описывает входные данные для создания Lot.
type CreateLotDTO struct {
	LotID           string    `json:"lot_id"`
	ProductID       string    `json:"product_id"`
	SellerCompanyID string    `json:"seller_company_id"`
	Photo           string    `json:"photo,omitempty"`
	Quantity        float64   `json:"quantity"`
	StartPrice      int64     `json:"start_price"`
	AuctionStartsAt time.Time `json:"auction_starts_at"`
}

// AssignAuctionIDDTO используется для привязки auctionID к Lot.
type AssignAuctionIDDTO struct {
	LotID     string `json:"lot_id"`
	AuctionID string `json:"auction_id"`
}

// PublishLotDTO используется для публикации Lot.
type PublishLotDTO struct {
	LotID string `json:"lot_id"`
}

// UnpublishLotDTO используется для снятия Lot с публикации.
type UnpublishLotDTO struct {
	LotID string `json:"lot_id"`
}

// CloseLotDTO используется для закрытия Lot с финальной ценой.
type CloseLotDTO struct {
	LotID      string `json:"lot_id"`
	FinalPrice int64  `json:"final_price"`
}

// FishDTO представляет Fish на выходе из application/transport слоя.
type FishDTO struct {
	FishID      string `json:"fish_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ProductDTO представляет Product на выходе.
type ProductDTO struct {
	ProductID      string           `json:"product_id"`
	FishID         string           `json:"fish_id"`
	Weight         float64          `json:"weight"`
	Unit           string           `json:"unit"`
	Size           string           `json:"size,omitempty"`
	ProcessingType string           `json:"processing_type"`
	Status         ProductStatusDTO `json:"status"`
}

// ProductSnapshotDTO представляет срез Product внутри integration events.
type ProductSnapshotDTO struct {
	FishID         string  `json:"fish_id"`
	Weight         float64 `json:"weight"`
	Unit           string  `json:"unit"`
	Size           string  `json:"size,omitempty"`
	ProcessingType string  `json:"processing_type"`
}

// LotDTO представляет Lot на выходе из application/transport слоя.
type LotDTO struct {
	LotID           string       `json:"lot_id"`
	ProductID       string       `json:"product_id"`
	AuctionID       string       `json:"auction_id,omitempty"`
	SellerCompanyID string       `json:"seller_company_id"`
	Photo           string       `json:"photo,omitempty"`
	Quantity        float64      `json:"quantity"`
	StartPrice      int64        `json:"start_price"`
	CurPrice        int64        `json:"cur_price"`
	FinalPrice      int64        `json:"final_price"`
	Status          LotStatusDTO `json:"status"`
	AuctionStartsAt time.Time    `json:"auction_starts_at"`
}

// ProductCreatedDTO публикуется после создания Product.
type ProductCreatedDTO struct {
	Metadata       EventMetadataDTO `json:"metadata"`
	ProductID      string           `json:"product_id"`
	FishID         string           `json:"fish_id"`
	Weight         float64          `json:"weight"`
	Unit           string           `json:"unit"`
	Size           string           `json:"size,omitempty"`
	ProcessingType string           `json:"processing_type"`
	Status         ProductStatusDTO `json:"status"`
}

// ProductUpdatedDTO публикуется после изменения Product.
type ProductUpdatedDTO struct {
	Metadata       EventMetadataDTO `json:"metadata"`
	ProductID      string           `json:"product_id"`
	FishID         string           `json:"fish_id"`
	Weight         float64          `json:"weight"`
	Unit           string           `json:"unit"`
	Size           string           `json:"size,omitempty"`
	ProcessingType string           `json:"processing_type"`
	Status         ProductStatusDTO `json:"status"`
}

// ProductPublishedDTO публикуется после публикации Product.
type ProductPublishedDTO struct {
	Metadata  EventMetadataDTO `json:"metadata"`
	ProductID string           `json:"product_id"`
	Status    ProductStatusDTO `json:"status"`
}

// ProductUnpublishedDTO публикуется после снятия Product с публикации.
type ProductUnpublishedDTO struct {
	Metadata  EventMetadataDTO `json:"metadata"`
	ProductID string           `json:"product_id"`
	Status    ProductStatusDTO `json:"status"`
}

// LotCreatedDTO публикуется после создания Lot.
type LotCreatedDTO struct {
	Metadata        EventMetadataDTO `json:"metadata"`
	LotID           string           `json:"lot_id"`
	ProductID       string           `json:"product_id"`
	SellerCompanyID string           `json:"seller_company_id"`
	Photo           string           `json:"photo,omitempty"`
	Quantity        float64          `json:"quantity"`
	Status          LotStatusDTO     `json:"status"`
}

// LotPublishedDTO публикуется после публикации Lot.
type LotPublishedDTO struct {
	Metadata        EventMetadataDTO   `json:"metadata"`
	LotID           string             `json:"lot_id"`
	AuctionID       string             `json:"auction_id"`
	SellerCompanyID string             `json:"seller_company_id"`
	ProductID       string             `json:"product_id"`
	Product         ProductSnapshotDTO `json:"product"`
	StartPrice      int64              `json:"start_price"`
	Status          LotStatusDTO       `json:"status"`
}

// LotUnpublishedDTO публикуется после снятия Lot с публикации.
type LotUnpublishedDTO struct {
	Metadata EventMetadataDTO `json:"metadata"`
	LotID    string           `json:"lot_id"`
	Status   LotStatusDTO     `json:"status"`
}

// LotClosedDTO публикуется после закрытия Lot.
type LotClosedDTO struct {
	Metadata   EventMetadataDTO `json:"metadata"`
	LotID      string           `json:"lot_id"`
	FinalPrice int64            `json:"final_price"`
	Status     LotStatusDTO     `json:"status"`
}

// LotPriceUpdatedDTO можно использовать как DTO для проекций/UI,
// когда нужно передать обновлённую текущую цену лота.
type LotPriceUpdatedDTO struct {
	Metadata  EventMetadataDTO `json:"metadata"`
	LotID     string           `json:"lot_id"`
	AuctionID string           `json:"auction_id,omitempty"`
	CurPrice  int64            `json:"cur_price"`
	Status    LotStatusDTO     `json:"status"`
}

// AuctionWonDTO описывает событие Trading, на которое реагирует Catalog.
type AuctionWonDTO struct {
	Metadata        EventMetadataDTO `json:"metadata"`
	AuctionID       string           `json:"auction_id"`
	WinnerCompanyID string           `json:"winner_company_id"`
	FinalPrice      int64            `json:"final_price"`
}

// BidPlacedDTO описывает событие Trading о новой ставке.
type BidPlacedDTO struct {
	Metadata  EventMetadataDTO `json:"metadata"`
	AuctionID string           `json:"auction_id"`
	Amount    int64            `json:"amount"`
}

// AuctionClosedDTO описывает событие Trading о закрытии аукциона.
type AuctionClosedDTO struct {
	Metadata  EventMetadataDTO `json:"metadata"`
	AuctionID string           `json:"auction_id"`
}

// AuctionCancelledDTO описывает событие Trading об отмене аукциона.
type AuctionCancelledDTO struct {
	Metadata  EventMetadataDTO `json:"metadata"`
	AuctionID string           `json:"auction_id"`
}
