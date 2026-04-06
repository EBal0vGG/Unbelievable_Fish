package handler

import (
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type PublishAuctionHandler struct {
	uc *app.PublishAuction
}

func NewPublishAuctionHandler(uc *app.PublishAuction) *PublishAuctionHandler {
	return &PublishAuctionHandler{uc: uc}
}

func (h *PublishAuctionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !requirePost(w, r, app.CommandMeta{}) {
		return
	}
	meta, err := readCommandMeta(r)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	auctionID, err := readAuctionIDFromPath(r.URL.Path, "publish")
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
