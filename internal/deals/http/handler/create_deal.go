package handler

import (
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
)

type CreateDealFromAuctionWonHandler struct {
	uc *app.CreateDealSelectionFromAuctionWon
}

func NewCreateDealFromAuctionWonHandler(uc *app.CreateDealSelectionFromAuctionWon) *CreateDealFromAuctionWonHandler {
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
	if err := h.uc.Execute(r.Context(), meta, req.AuctionID, []string{req.WinnerCompanyID}, req.FinalPrice, req.WonAt); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}
