package handler

import (
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type CloseAuctionHandler struct {
	uc *app.CloseAuction
}

func NewCloseAuctionHandler(uc *app.CloseAuction) *CloseAuctionHandler {
	return &CloseAuctionHandler{uc: uc}
}

func (h *CloseAuctionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	if !requirePost(w, r, meta) {
		return
	}
	auctionID, err := readAuctionIDFromPath(r.URL.Path, "close")
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
