package handler

import (
	"net/http"

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
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	meta, err := readCommandMeta(r)
	if err != nil {
		httpErr := httpapi.MapError(err)
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	auctionID, err := readAuctionIDFromPath(r.URL.Path, "bids")
	if err != nil {
		httpErr := httpapi.MapError(err)
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	var req httpapi.PlaceBidRequest
	if err := decodeJSON(r, &req); err != nil {
		httpErr := httpapi.BadRequest("INVALID_BODY", "invalid request body")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	if req.Amount == 0 || req.PlacedAt.IsZero() {
		httpErr := httpapi.BadRequest("INVALID_BODY", "amount and placed_at are required")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, auctionID, req.Amount, req.PlacedAt); err != nil {
		httpErr := httpapi.MapError(err)
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
