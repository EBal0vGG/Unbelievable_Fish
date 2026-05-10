package handler

import (
	"net/http"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
)

type PlaceBidHandler struct {
	uc *app.PlaceBid
}

func NewPlaceBidHandler(uc *app.PlaceBid) *PlaceBidHandler {
	return &PlaceBidHandler{uc: uc}
}

func (h *PlaceBidHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	auctionID, err := readAuctionIDFromRequest(r)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	var req httpapi.PlaceBidRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httpErr := httpapi.BadRequest("INVALID_BODY", "invalid request body")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	if req.Amount <= 0 {
		httpErr := httpapi.BadRequest("INVALID_BODY", "amount is required")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	placedAt := time.Now().UTC()
	if !req.PlacedAt.IsZero() {
		placedAt = req.PlacedAt.UTC()
	}
	result, err := h.uc.ExecuteWithResult(r.Context(), meta, auctionID, req.Amount, placedAt)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	if result == nil {
		writeAccepted(w)
		return
	}
	writeAcceptedJSON(w, httpapi.PlaceBidResponse{
		AuctionID: string(result.AuctionID),
		Chain: httpapi.PlaceBidChainDTO{
			BidHash:       result.BidHash,
			TxHash:        result.ChainTxHash,
			Status:        result.ChainStatus,
			WalletAddress: result.ChainWalletAddress,
		},
	})
}
