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

type GetDealConfirmationsHandler struct {
	uc *app.GetDealConfirmations
}

func NewGetDealConfirmationsHandler(uc *app.GetDealConfirmations) *GetDealConfirmationsHandler {
	return &GetDealConfirmationsHandler{uc: uc}
}

func (h *GetDealConfirmationsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, app.CommandMeta{})
		return
	}
	items, err := h.uc.Execute(r.Context(), dealID)
	if err != nil {
		writeError(w, err, app.CommandMeta{})
		return
	}
	response := make([]httpapi.DealConfirmationResponse, 0, len(items))
	for _, item := range items {
		response = append(response, httpapi.NewDealConfirmationResponse(item))
	}
	writeJSON(w, http.StatusOK, response)
}
