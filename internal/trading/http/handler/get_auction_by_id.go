package handler

import (
	"encoding/json"
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type GetAuctionByIDHandler struct {
	uc *app.GetAuctionByID
}

func NewGetAuctionByIDHandler(uc *app.GetAuctionByID) *GetAuctionByIDHandler {
	return &GetAuctionByIDHandler{uc: uc}
}

func (h *GetAuctionByIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	auctionID, err := readAuctionIDFromRequest(r)
	if err != nil {
		handleCommandError(w, err, app.CommandMeta{})
		return
	}
	result, err := h.uc.Execute(r.Context(), auctionID)
	if err != nil {
		handleCommandError(w, err, app.CommandMeta{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(auctionSummaryResponse(result))
}
