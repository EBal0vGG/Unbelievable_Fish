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
	var req httpapi.CreateAuctionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body", meta)
		return
	}
	if req.AuctionID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_BODY", "auction_id is required", meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, app.AuctionID(req.AuctionID)); err != nil {
		status, code, message := mapError(err)
		writeError(w, status, code, message, meta)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
