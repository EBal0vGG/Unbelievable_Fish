package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	"github.com/go-chi/chi/v5"
)

func dealInvoiceJSON(inv *wallet.DealInvoice) map[string]any {
	m := map[string]any{
		"id":                      inv.ID,
		"deal_id":                 inv.DealID,
		"auction_id":              inv.AuctionID,
		"buyer_company_id":        inv.BuyerCompanyID,
		"seller_company_id":       inv.SellerCompanyID,
		"goods_amount":            inv.GoodsAmount,
		"platform_fee_due_amount": inv.PlatformFeeDueAmount,
		"total_amount":            inv.TotalAmount,
		"currency":                string(inv.Currency),
		"status":                  string(inv.Status),
		"provider":                inv.Provider,
		"provider_invoice_id":     inv.ProviderInvoiceID,
		"payment_url":             inv.PaymentURL,
		"due_at":                  inv.DueAt,
		"created_at":              inv.CreatedAt,
	}
	if inv.PaidAt != nil {
		m["paid_at"] = *inv.PaidAt
	}
	if inv.ExpiredAt != nil {
		m["expired_at"] = *inv.ExpiredAt
	}
	if inv.CancelledAt != nil {
		m["cancelled_at"] = *inv.CancelledAt
	}
	if inv.FailedAt != nil {
		m["failed_at"] = *inv.FailedAt
	}
	return m
}

type GetDealInvoiceHandler struct {
	invoices billingapp.DealInvoiceRepository
}

func NewGetDealInvoiceHandler(repo billingapp.DealInvoiceRepository) *GetDealInvoiceHandler {
	return &GetDealInvoiceHandler{invoices: repo}
}

func (h *GetDealInvoiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	companyID, ok := identityauth.CompanyIDFromContext(r.Context())
	if !ok || companyID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	invoiceID := chi.URLParam(r, "invoiceID")
	if invoiceID == "" {
		http.Error(w, `{"error":"MISSING_INVOICE_ID"}`, http.StatusBadRequest)
		return
	}
	inv, err := h.invoices.LoadByID(r.Context(), invoiceID)
	if err != nil {
		if errors.Is(err, billingapp.ErrDealInvoiceNotFound) {
			http.Error(w, `{"error":"INVOICE_NOT_FOUND"}`, http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if inv.BuyerCompanyID != companyID && inv.SellerCompanyID != companyID {
		http.Error(w, `{"error":"FORBIDDEN"}`, http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dealInvoiceJSON(inv))
}

type GetDealInvoiceByDealHandler struct {
	invoices billingapp.DealInvoiceRepository
}

func NewGetDealInvoiceByDealHandler(repo billingapp.DealInvoiceRepository) *GetDealInvoiceByDealHandler {
	return &GetDealInvoiceByDealHandler{invoices: repo}
}

func (h *GetDealInvoiceByDealHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	companyID, ok := identityauth.CompanyIDFromContext(r.Context())
	if !ok || companyID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	dealID := chi.URLParam(r, "dealID")
	if dealID == "" {
		http.Error(w, `{"error":"MISSING_DEAL_ID"}`, http.StatusBadRequest)
		return
	}
	inv, err := h.invoices.LoadByDealID(r.Context(), dealID)
	if err != nil {
		if errors.Is(err, billingapp.ErrDealInvoiceNotFound) {
			http.Error(w, `{"error":"INVOICE_NOT_FOUND"}`, http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if inv.BuyerCompanyID != companyID && inv.SellerCompanyID != companyID {
		http.Error(w, `{"error":"FORBIDDEN"}`, http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dealInvoiceJSON(inv))
}

type ListMyDealInvoicesHandler struct {
	invoices billingapp.DealInvoiceRepository
}

func NewListMyDealInvoicesHandler(repo billingapp.DealInvoiceRepository) *ListMyDealInvoicesHandler {
	return &ListMyDealInvoicesHandler{invoices: repo}
}

func (h *ListMyDealInvoicesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	list, err := h.invoices.ListByBuyerCompany(r.Context(), companyID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, inv := range list {
		out = append(out, dealInvoiceJSON(inv))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"invoices": out})
}

type FakeConfirmDealInvoiceHandler struct {
	tx TxRunner
	uc *billingapp.ConfirmDealInvoicePaid
}

func NewFakeConfirmDealInvoiceHandler(tx TxRunner, uc *billingapp.ConfirmDealInvoicePaid) *FakeConfirmDealInvoiceHandler {
	return &FakeConfirmDealInvoiceHandler{tx: tx, uc: uc}
}

func (h *FakeConfirmDealInvoiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	invoiceID := chi.URLParam(r, "invoiceID")
	if invoiceID == "" {
		http.Error(w, `{"error":"MISSING_INVOICE_ID"}`, http.StatusBadRequest)
		return
	}
	if err := h.tx.WithinTx(r.Context(), func(ctx context.Context) error {
		return h.uc.Execute(ctx, invoiceID)
	}); err != nil {
		if errors.Is(err, billingapp.ErrDealInvoiceNotFound) {
			http.Error(w, `{"error":"INVOICE_NOT_FOUND"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, wallet.ErrInvoiceNotPayable) || errors.Is(err, wallet.ErrInvoiceAmountMismatch) || errors.Is(err, wallet.ErrInvoiceCurrencyMismatch) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
