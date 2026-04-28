package httpapi

import "time"

// DTOs for external API contract (commands).
type CreateAuctionRequest struct {
	LotID      string    `json:"lot_id"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	StartPrice int64     `json:"start_price,omitempty"`
	MinBidStep int64     `json:"min_bid_step,omitempty"`
}

type CreateAuctionResponse struct {
	AuctionID string `json:"auction_id"`
}

type PlaceBidRequest struct {
	Amount   int64     `json:"amount"`
	PlacedAt time.Time `json:"placed_at"`
}

type ErrorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
}
