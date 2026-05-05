package handler

import (
	"encoding/json"
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type GetAuctionByLotHandler struct {
	uc *app.GetAuctionByLot
}

func NewGetAuctionByLotHandler(uc *app.GetAuctionByLot) *GetAuctionByLotHandler {
	return &GetAuctionByLotHandler{uc: uc}
}

func (h *GetAuctionByLotHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lotID, err := readLotIDFromRequest(r)
	if err != nil {
		handleCommandError(w, err, app.CommandMeta{})
		return
	}
	result, err := h.uc.Execute(r.Context(), lotID)
	if err != nil {
		handleCommandError(w, err, app.CommandMeta{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(auctionSummaryResponse(result))
}
