package handler

import (
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
)

type GetProjectionByAuctionIDHandler struct {
	uc *app.GetProjectionByAuctionID
}

func NewGetProjectionByAuctionIDHandler(uc *app.GetProjectionByAuctionID) *GetProjectionByAuctionIDHandler {
	return &GetProjectionByAuctionIDHandler{uc: uc}
}

func (h *GetProjectionByAuctionIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	auctionID, err := readAuctionIDFromRequest(r)
	if err != nil {
		writeError(w, err, app.CommandMeta{})
		return
	}
	item, err := h.uc.Execute(r.Context(), auctionID)
	if err != nil {
		writeError(w, err, app.CommandMeta{})
		return
	}
	writeJSON(w, http.StatusOK, httpapi.NewProjectionResponse(item))
}

type GetDealByIDHandler struct {
	uc *app.GetDealByID
}

func NewGetDealByIDHandler(uc *app.GetDealByID) *GetDealByIDHandler {
	return &GetDealByIDHandler{uc: uc}
}

func (h *GetDealByIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, app.CommandMeta{})
		return
	}
	item, err := h.uc.Execute(r.Context(), dealID)
	if err != nil {
		writeError(w, err, app.CommandMeta{})
		return
	}
	writeJSON(w, http.StatusOK, httpapi.NewDealResponse(item))
}

type GetDealByAuctionIDHandler struct {
	uc *app.GetDealByAuctionID
}

func NewGetDealByAuctionIDHandler(uc *app.GetDealByAuctionID) *GetDealByAuctionIDHandler {
	return &GetDealByAuctionIDHandler{uc: uc}
}

func (h *GetDealByAuctionIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	auctionID, err := readAuctionIDFromRequest(r)
	if err != nil {
		writeError(w, err, app.CommandMeta{})
		return
	}
	item, err := h.uc.Execute(r.Context(), auctionID)
	if err != nil {
		writeError(w, err, app.CommandMeta{})
		return
	}
	writeJSON(w, http.StatusOK, httpapi.NewDealResponse(item))
}
