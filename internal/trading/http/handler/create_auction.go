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
		httpErr := httpapi.MapError(err)
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	var req httpapi.CreateAuctionRequest
	if err := decodeJSON(r, &req); err != nil {
		httpErr := httpapi.BadRequest("INVALID_BODY", "invalid request body")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	if req.AuctionID == "" {
		httpErr := httpapi.BadRequest("INVALID_BODY", "auction_id is required")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, app.AuctionID(req.AuctionID)); err != nil {
		httpErr := httpapi.MapError(err)
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, meta)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
