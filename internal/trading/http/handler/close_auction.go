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
	auctionID, err := readAuctionIDFromPath(r.URL.Path, "close")
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
