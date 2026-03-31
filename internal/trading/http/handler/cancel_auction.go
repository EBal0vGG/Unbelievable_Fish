package handler

import (
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type CancelAuctionHandler struct {
	uc *app.CancelAuction
}

func NewCancelAuctionHandler(uc *app.CancelAuction) *CancelAuctionHandler {
	return &CancelAuctionHandler{uc: uc}
}

func (h *CancelAuctionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	if !requirePost(w, r, meta) {
		return
	}
	auctionID, err := readAuctionIDFromPath(r.URL.Path, "cancel")
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, auctionID); err != nil {
		handleCommandError(w, err, meta)
		return
	}
	writeAccepted(w)
}
