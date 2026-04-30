package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
)

type GetAuctionByIDHandler struct {
	uc *app.GetAuctionByID
}

func NewGetAuctionByIDHandler(uc *app.GetAuctionByID) *GetAuctionByIDHandler {
	return &GetAuctionByIDHandler{uc: uc}
}

func (h *GetAuctionByIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpErr := httpapi.MethodNotAllowed("METHOD_NOT_ALLOWED", "method not allowed")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, app.CommandMeta{})
		return
	}
	auctionID, err := readAuctionIDQueryPath(r.URL.Path)
	if err != nil {
		handleCommandError(w, err, app.CommandMeta{})
		return
	}
	result, err := h.uc.Execute(r.Context(), auctionID)
	if err != nil {
		handleCommandError(w, err, app.CommandMeta{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"auction_id":        result.AuctionID,
		"lot_id":            result.LotID,
		"state":             result.State,
		"starts_at":         result.StartsAt,
		"ends_at":           result.EndsAt,
		"current_price":     result.CurrentPrice,
		"min_bid_step":      result.MinBidStep,
		"leader_company_id": result.LeaderCompanyID,
	})
}

func readAuctionIDQueryPath(path string) (app.AuctionID, error) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] != "auctions" || parts[1] == "" {
		return "", httpapi.ErrInvalidPath
	}
	return app.AuctionID(parts[1]), nil
}
