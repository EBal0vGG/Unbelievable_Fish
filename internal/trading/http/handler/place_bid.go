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
	if !requirePost(w, r, app.CommandMeta{}) {
		return
	}
	meta, err := readCommandMeta(r)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	auctionID, err := readAuctionIDFromPath(r.URL.Path, "bids")
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
	if err := h.uc.Execute(r.Context(), meta, auctionID, req.Amount, placedAt); err != nil {
		handleCommandError(w, err, meta)
		return
	}
	writeAccepted(w)
}
