package httpapi

import "time"

type CreateAuctionRequest struct {
	AuctionID string `json:"auction_id"`
}

type PlaceBidRequest struct {
	BidderCompanyID string    `json:"bidder_company_id"`
	Amount          int64     `json:"amount"`
	PlacedAt        time.Time `json:"placed_at"`
}

type ErrorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
}
