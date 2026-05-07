package handler

import (
	"encoding/json"
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type ListAuctionsHandler struct {
	uc *app.ListAuctions
}

func NewListAuctionsHandler(uc *app.ListAuctions) *ListAuctionsHandler {
	return &ListAuctionsHandler{uc: uc}
}

func (h *ListAuctionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result, err := h.uc.Execute(r.Context())
	if err != nil {
		handleCommandError(w, err, app.CommandMeta{})
		return
	}
	out := make([]map[string]any, 0, len(result))
	for _, item := range result {
		out = append(out, auctionSummaryResponse(item))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

func auctionSummaryResponse(result *app.AuctionSummary) map[string]any {
	return map[string]any{
		"auction_id":        result.AuctionID,
		"lot_id":            result.LotID,
		"state":             result.State,
		"starts_at":         result.StartsAt,
		"ends_at":           result.EndsAt,
		"start_price":       result.StartPrice,
		"current_price":     result.CurrentPrice,
		"min_bid_step":      result.MinBidStep,
		"leader_company_id": result.LeaderCompanyID,
		"winner_company_id": winnerCompanyID(result),
		"final_price":       finalPrice(result),
	}
}

func winnerCompanyID(result *app.AuctionSummary) string {
	if result.State != "WON" && result.State != "CLOSED" {
		return ""
	}
	return result.LeaderCompanyID
}

func finalPrice(result *app.AuctionSummary) int64 {
	if result.State != "WON" && result.State != "CLOSED" {
		return 0
	}
	return result.CurrentPrice
}
