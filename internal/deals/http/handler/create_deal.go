package handler

import (
	"net/http"

	"unbelievable_fish/internal/deals/app"
	httpapi "unbelievable_fish/internal/deals/http"
)

type CreateDealFromAuctionWonHandler struct {
	uc *app.CreateDealFromAuctionWon
}

func NewCreateDealFromAuctionWonHandler(uc *app.CreateDealFromAuctionWon) *CreateDealFromAuctionWonHandler {
	return &CreateDealFromAuctionWonHandler{uc: uc}
}

func (h *CreateDealFromAuctionWonHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.CreateDealFromAuctionWonRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, req.AuctionID, req.WinnerCompanyID, req.FinalPrice, req.WonAt); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}
