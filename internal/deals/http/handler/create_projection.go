package handler

import (
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
)

type CreateProjectionHandler struct {
	uc *app.CreateProjection
}

func NewCreateProjectionHandler(uc *app.CreateProjection) *CreateProjectionHandler {
	return &CreateProjectionHandler{uc: uc}
}

func (h *CreateProjectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.CreateProjectionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, req.AuctionID, req.SupplierID, req.ProductSnapshot.ToDomain(), req.StartPrice, req.PublishedAt); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}
