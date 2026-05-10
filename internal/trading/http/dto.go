package httpapi

import "time"

type PlaceBidRequest struct {
	Amount   int64     `json:"amount"`
	PlacedAt time.Time `json:"placed_at"`
}

type PlaceBidChainDTO struct {
	BidHash       string `json:"bid_hash,omitempty"`
	TxHash        string `json:"tx_hash,omitempty"`
	Status        string `json:"status,omitempty"`
	WalletAddress string `json:"wallet_address,omitempty"`
}

type PlaceBidResponse struct {
	AuctionID string           `json:"auction_id"`
	Chain     PlaceBidChainDTO `json:"chain"`
}

type ErrorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
}
