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
		status, code, message := mapError(err)
		writeError(w, status, code, message, meta)
		return
	}
	auctionID, err := readAuctionIDFromPath(r.URL.Path, "bids")
	if err != nil {
		status, code, message := mapError(err)
		writeError(w, status, code, message, meta)
		return
	}
	var req httpapi.PlaceBidRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body", meta)
		return
	}
	if req.BidderCompanyID == "" || req.Amount == 0 || req.PlacedAt.IsZero() {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "bidder_company_id, amount and placed_at are required", meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, auctionID, req.BidderCompanyID, req.Amount, req.PlacedAt); err != nil {
		status, code, message := mapError(err)
		writeError(w, status, code, message, meta)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
