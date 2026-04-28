package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
)

type GetAuctionByLotHandler struct {
	uc *app.GetAuctionByLot
}

func NewGetAuctionByLotHandler(uc *app.GetAuctionByLot) *GetAuctionByLotHandler {
	return &GetAuctionByLotHandler{uc: uc}
}

func (h *GetAuctionByLotHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpErr := httpapi.MethodNotAllowed("METHOD_NOT_ALLOWED", "method not allowed")
		writeError(w, httpErr.Status, httpErr.Code, httpErr.Message, app.CommandMeta{})
		return
	}
	lotID, err := readLotIDFromPath(r.URL.Path)
	if err != nil {
		handleCommandError(w, err, app.CommandMeta{})
		return
	}
	result, err := h.uc.Execute(r.Context(), lotID)
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
		"current_price":     result.CurrentPrice,
		"leader_company_id": result.LeaderCompanyID,
	})
}

func readLotIDFromPath(path string) (string, error) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 3 || parts[0] != "auctions" || parts[1] != "by-lot" || parts[2] == "" {
		return "", httpapi.ErrInvalidPath
	}
	return parts[2], nil
}
