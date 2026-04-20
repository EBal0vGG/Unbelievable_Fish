package handler

import (
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
)

type CreateAuctionHandler struct {
	uc *app.CreateAuction
}

func NewCreateAuctionHandler(uc *app.CreateAuction) *CreateAuctionHandler {
	return &CreateAuctionHandler{uc: uc}
}

func (h *CreateAuctionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r, app.CommandMeta{}) {
		return
	}
	meta, err := readCommandMeta(r)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	var req httpapi.CreateAuctionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		httpErr := httpapi.BadRequest("INVALID_BODY", "invalid request body")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	if req.LotID == "" {
		httpErr := httpapi.BadRequest("INVALID_BODY", "lot_id is required")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	if req.StartsAt.IsZero() || req.EndsAt.IsZero() {
		httpErr := httpapi.BadRequest("INVALID_BODY", "starts_at and ends_at are required")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	auctionID, err := h.uc.Execute(r.Context(), meta, req.LotID, req.StartsAt, req.EndsAt)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	writeAcceptedJSON(w, httpapi.CreateAuctionResponse{AuctionID: string(auctionID)})
}
