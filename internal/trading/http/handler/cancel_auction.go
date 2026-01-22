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
	auctionID, err := readAuctionIDFromPath(r.URL.Path, "cancel")
	if err != nil {
		status, code, message := mapError(err)
		writeError(w, status, code, message, meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, auctionID); err != nil {
		status, code, message := mapError(err)
		writeError(w, status, code, message, meta)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
