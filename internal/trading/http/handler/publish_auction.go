package handler

import (
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
)

type PublishAuctionHandler struct {
	uc *app.PublishAuction
}

func NewPublishAuctionHandler(uc *app.PublishAuction) *PublishAuctionHandler {
	return &PublishAuctionHandler{uc: uc}
}

func (h *PublishAuctionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	auctionID, err := readAuctionIDFromPath(r.URL.Path, "publish")
	if err != nil {
		httpErr := httpapi.MapError(err)
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, auctionID); err != nil {
		httpErr := httpapi.MapError(err)
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
