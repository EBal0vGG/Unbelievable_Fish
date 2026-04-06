package dto

import "time"

// RequestMetadataDTO содержит транспортные метаданные запроса.
type RequestMetadataDTO struct {
	CompanyID     string `json:"company_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
}

// EventMetadataDTO содержит метаданные event envelope для outbox/broker.
type EventMetadataDTO struct {
	EventID       string     `json:"event_id,omitempty"`
	EventType     string     `json:"event_type,omitempty"`
	EventVersion  string     `json:"event_version,omitempty"`
	OccurredAt    *time.Time `json:"occurred_at,omitempty"`
	CorrelationID string     `json:"correlation_id,omitempty"`
	CausationID   string     `json:"causation_id,omitempty"`
	Producer      string     `json:"producer,omitempty"`
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
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// UpdateFishDTO используется для обновления Fish.
// Идентификатор Fish должен приходить из path params, а не из body.
type UpdateFishDTO struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreateProductDTO описывает входные данные для создания Product.
type CreateProductDTO struct {
	FishID         string  `json:"fish_id"`
	Weight         float64 `json:"weight"`
	Unit           string  `json:"unit"`
	Size           string  `json:"size,omitempty"`
	ProcessingType string  `json:"processing_type"`
}

// UpdateProductDTO описывает входные данные для изменения Product.
// Идентификатор Product должен приходить из path params, а не из body.
type UpdateProductDTO struct {
	FishID         string  `json:"fish_id"`
	Weight         float64 `json:"weight"`
	Unit           string  `json:"unit"`
	Size           string  `json:"size,omitempty"`
	ProcessingType string  `json:"processing_type"`
}

// PublishProductDTO используется для публикации Product.
// Идентификатор Product должен приходить из path params; body не требуется.
type PublishProductDTO struct{}

// UnpublishProductDTO используется для снятия Product с публикации.
// Идентификатор Product должен приходить из path params; body не требуется.
type UnpublishProductDTO struct{}

// CreateLotDTO описывает входные данные для создания Lot.
type CreateLotDTO struct {
	ProductID       string    `json:"product_id"`
	Photo           string    `json:"photo,omitempty"`
	Quantity        float64   `json:"quantity"`
	StartPrice      int64     `json:"start_price"`
	AuctionStartsAt time.Time `json:"auction_starts_at"`
}

// AssignAuctionIDDTO используется для привязки auctionID к Lot.
// Это DTO для integration event, а не для публичного HTTP body.
type AssignAuctionIDDTO struct {
	LotID     string `json:"lot_id"`
	AuctionID string `json:"auction_id"`
}

// PublishLotDTO используется для публикации Lot.
// Идентификатор Lot должен приходить из path params; body не требуется.
type PublishLotDTO struct{}

// UnpublishLotDTO используется для снятия Lot с публикации.
// Идентификатор Lot должен приходить из path params; body не требуется.
type UnpublishLotDTO struct{}

// CloseLotDTO используется для закрытия Lot с финальной ценой.
// Идентификатор Lot должен приходить из path params.
// Используется только для административного override-сценария, если он существует.
type CloseLotDTO struct {
	FinalPrice int64 `json:"final_price"`
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
	AuctionID       *string      `json:"auction_id,omitempty"`
	SellerCompanyID string       `json:"seller_company_id"`
	Photo           string       `json:"photo,omitempty"`
	Quantity        float64      `json:"quantity"`
	StartPrice      int64        `json:"start_price"`
	CurrentPrice    *int64       `json:"current_price,omitempty"`
	FinalPrice      *int64       `json:"final_price,omitempty"`
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
	StartPrice      int64            `json:"start_price"`
	AuctionStartsAt time.Time        `json:"auction_starts_at"`
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

// LotPriceUpdatedDTO публикуется после обновления текущей цены Lot.
type LotPriceUpdatedDTO struct {
	Metadata     EventMetadataDTO `json:"metadata"`
	LotID        string           `json:"lot_id"`
	AuctionID    string           `json:"auction_id"`
	CurrentPrice int64            `json:"current_price"`
	Status       LotStatusDTO     `json:"status"`
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
