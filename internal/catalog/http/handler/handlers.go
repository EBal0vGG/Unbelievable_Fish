package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/http"
)

type ListFishHandler struct{ svc *catalogapp.CatalogService }

func NewListFishHandler(svc *catalogapp.CatalogService) *ListFishHandler {
	return &ListFishHandler{svc: svc}
}

func (h *ListFishHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListFish(r.Context())
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	type fishItem struct {
		ID          string `json:"id"`
		FishID      string `json:"fish_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	out := make([]fishItem, 0, len(list))
	for _, fish := range list {
		out = append(out, fishItem{
			ID:          fish.ID(),
			FishID:      fish.ID(),
			Name:        fish.Name(),
			Description: fish.Description(),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type CreateFishHandler struct{ svc *catalogapp.CatalogService }

func NewCreateFishHandler(svc *catalogapp.CatalogService) *CreateFishHandler {
	return &CreateFishHandler{svc: svc}
}

func (h *CreateFishHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPError(w, fmt.Errorf("%w: %v", httpapi.ErrInvalidJSONBody, err))
		return
	}
	id, err := h.svc.CreateFish(r.Context(), catalogapp.CreateFishCommand{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"fish_id": id})
}

type CreateProductHandler struct{ svc *catalogapp.CatalogService }

func NewCreateProductHandler(svc *catalogapp.CatalogService) *CreateProductHandler {
	return &CreateProductHandler{svc: svc}
}

func (h *CreateProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FishID         string  `json:"fish_id"`
		Weight         float64 `json:"weight"`
		Unit           string  `json:"unit"`
		Size           string  `json:"size"`
		ProcessingType string  `json:"processing_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPError(w, fmt.Errorf("%w: %v", httpapi.ErrInvalidJSONBody, err))
		return
	}
	id, _, err := h.svc.CreateProduct(catalogRequestContext(r), catalogapp.CreateProductCommand{
		FishID:         req.FishID,
		Weight:         req.Weight,
		Unit:           req.Unit,
		Size:           req.Size,
		ProcessingType: catalog.ProcessingType(req.ProcessingType),
	})
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"product_id": id})
}

type PublishProductHandler struct{ svc *catalogapp.CatalogService }

func NewPublishProductHandler(svc *catalogapp.CatalogService) *PublishProductHandler {
	return &PublishProductHandler{svc: svc}
}

func (h *PublishProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	productID, err := productIDFromRequest(r)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	if err := h.svc.PublishProduct(catalogRequestContext(r), productID); err != nil {
		writeHTTPError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type ListProductsHandler struct{ svc *catalogapp.CatalogService }

func NewListProductsHandler(svc *catalogapp.CatalogService) *ListProductsHandler {
	return &ListProductsHandler{svc: svc}
}

func (h *ListProductsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListProducts(catalogRequestContext(r))
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	type row struct {
		ProductID       string  `json:"product_id"`
		FishID          string  `json:"fish_id"`
		SellerCompanyID string  `json:"seller_company_id"`
		Weight          float64 `json:"weight"`
		Unit            string  `json:"unit"`
		Size            string  `json:"size"`
		ProcessingType  string  `json:"processing_type"`
		Status          string  `json:"status"`
	}
	out := make([]row, 0, len(list))
	for _, p := range list {
		out = append(out, row{
			ProductID:       p.ID(),
			FishID:          p.FishID(),
			SellerCompanyID: p.SellerCompanyID(),
			Weight:          p.Weight(),
			Unit:            p.Unit(),
			Size:            p.Size(),
			ProcessingType:  string(p.ProcessingType()),
			Status:          string(p.Status()),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type CreateLotHandler struct {
	svc *catalogapp.CatalogService
}

func NewCreateLotHandler(svc *catalogapp.CatalogService) *CreateLotHandler {
	return &CreateLotHandler{svc: svc}
}

func (h *CreateLotHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProductID              string    `json:"product_id"`
		Photo                  string    `json:"photo"`
		Quantity               float64   `json:"quantity"`
		StartPrice             int64     `json:"start_price"`
		MinBidStep             int64     `json:"min_bid_step"`
		AuctionStartsAt        time.Time `json:"auction_starts_at"`
		AuctionDurationMinutes int64     `json:"auction_duration_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPError(w, fmt.Errorf("%w: %v", httpapi.ErrInvalidJSONBody, err))
		return
	}
	durationMinutes := req.AuctionDurationMinutes
	if durationMinutes <= 0 {
		durationMinutes = envDurationMinutesInt64("CATALOG_AUCTION_DURATION_MINUTES", 60)
	}
	ctx := catalogRequestContext(r)
	id, _, err := h.svc.CreateLot(ctx, catalogapp.CreateLotCommand{
		ProductID:              req.ProductID,
		Photo:                  req.Photo,
		Quantity:               req.Quantity,
		StartPrice:             req.StartPrice,
		MinBidStep:             req.MinBidStep,
		AuctionStartsAt:        req.AuctionStartsAt,
		AuctionDurationMinutes: durationMinutes,
	})
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"lot_id": id})
}

type PublishLotHandler struct{ svc *catalogapp.CatalogService }

func NewPublishLotHandler(svc *catalogapp.CatalogService) *PublishLotHandler {
	return &PublishLotHandler{svc: svc}
}

func (h *PublishLotHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lotID, err := lotIDFromRequest(r)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	if err := h.svc.PublishLot(catalogRequestContext(r), lotID); err != nil {
		writeHTTPError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type ListLotsHandler struct{ svc *catalogapp.CatalogService }

func NewListLotsHandler(svc *catalogapp.CatalogService) *ListLotsHandler {
	return &ListLotsHandler{svc: svc}
}

func (h *ListLotsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListLots(catalogRequestContext(r))
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	type row struct {
		LotID           string    `json:"lot_id"`
		ProductID       string    `json:"product_id"`
		SellerCompanyID string    `json:"seller_company_id"`
		AuctionID       string    `json:"auction_id,omitempty"`
		Photo           string    `json:"photo,omitempty"`
		Quantity        float64   `json:"quantity"`
		StartPrice      int64     `json:"start_price"`
		MinBidStep      int64     `json:"min_bid_step"`
		CurrentPrice    int64     `json:"current_price"`
		FinalPrice      int64     `json:"final_price,omitempty"`
		Status          string    `json:"status"`
		AuctionStartsAt time.Time `json:"auction_starts_at"`
		AuctionEndsAt   time.Time `json:"auction_ends_at"`
	}
	out := make([]row, 0, len(list))
	for _, l := range list {
		out = append(out, row{
			LotID:           l.ID(),
			ProductID:       l.ProductID(),
			SellerCompanyID: l.SellerCompanyID(),
			AuctionID:       l.AuctionID(),
			Photo:           l.Photo(),
			Quantity:        l.Quantity(),
			StartPrice:      l.StartPrice(),
			MinBidStep:      l.MinBidStep(),
			CurrentPrice:    l.CurPrice(),
			FinalPrice:      l.FinalPrice(),
			Status:          string(l.Status()),
			AuctionStartsAt: l.AuctionStartsAt(),
			AuctionEndsAt:   l.AuctionEndsAt(),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func envDurationMinutesInt64(key string, def int64) int64 {
	if value := os.Getenv(key); value != "" {
		if minutes, err := strconv.ParseInt(value, 10, 64); err == nil && minutes > 0 {
			return minutes
		}
	}
	return def
}
