package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	"github.com/go-chi/chi/v5"
)

func sellerPayoutJSON(p *wallet.SellerPayout) map[string]any {
	m := map[string]any{
		"payout_id":         p.ID,
		"deal_id":           p.DealID,
		"invoice_id":        p.InvoiceID,
		"auction_id":        p.AuctionID,
		"seller_company_id": p.SellerCompanyID,
		"buyer_company_id":  p.BuyerCompanyID,
		"amount":            p.Amount,
		"currency":          string(p.Currency),
		"status":            string(p.Status),
		"created_at":        p.CreatedAt,
	}
	if p.ReadyAt != nil {
		m["ready_at"] = *p.ReadyAt
	}
	if p.PaidAt != nil {
		m["paid_at"] = *p.PaidAt
	}
	if p.CancelledAt != nil {
		m["cancelled_at"] = *p.CancelledAt
	}
	if p.FailedAt != nil {
		m["failed_at"] = *p.FailedAt
	}
	return m
}

type ListMySellerPayoutsHandler struct {
	payouts billingapp.SellerPayoutRepository
}

func NewListMySellerPayoutsHandler(repo billingapp.SellerPayoutRepository) *ListMySellerPayoutsHandler {
	return &ListMySellerPayoutsHandler{payouts: repo}
}

func (h *ListMySellerPayoutsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	companyID, ok := identityauth.CompanyIDFromContext(r.Context())
	if !ok || companyID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	list, err := h.payouts.ListBySellerCompany(r.Context(), companyID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, p := range list {
		out = append(out, sellerPayoutJSON(p))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"payouts": out})
}

type GetSellerPayoutHandler struct {
	payouts billingapp.SellerPayoutRepository
}

func NewGetSellerPayoutHandler(repo billingapp.SellerPayoutRepository) *GetSellerPayoutHandler {
	return &GetSellerPayoutHandler{payouts: repo}
}

func (h *GetSellerPayoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	companyID, ok := identityauth.CompanyIDFromContext(r.Context())
	if !ok || companyID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	payoutID := chi.URLParam(r, "payoutID")
	if payoutID == "" {
		http.Error(w, `{"error":"MISSING_PAYOUT_ID"}`, http.StatusBadRequest)
		return
	}
	p, err := h.payouts.LoadByID(r.Context(), payoutID)
	if err != nil {
		if errors.Is(err, billingapp.ErrSellerPayoutNotFound) {
			http.Error(w, `{"error":"PAYOUT_NOT_FOUND"}`, http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if p.SellerCompanyID != companyID {
		http.Error(w, `{"error":"FORBIDDEN"}`, http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sellerPayoutJSON(p))
}
