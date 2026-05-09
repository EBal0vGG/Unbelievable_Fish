package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	"github.com/go-chi/chi/v5"
)

// AdminConfirmDealInvoiceHandler runs the same paid confirmation as fake-confirm (operator tooling).
type AdminConfirmDealInvoiceHandler struct {
	tx TxRunner
	uc *billingapp.ConfirmDealInvoicePaid
}

// NewAdminConfirmDealInvoiceHandler wires ConfirmDealInvoicePaid inside a billing transaction.
func NewAdminConfirmDealInvoiceHandler(tx TxRunner, uc *billingapp.ConfirmDealInvoicePaid) *AdminConfirmDealInvoiceHandler {
	return &AdminConfirmDealInvoiceHandler{tx: tx, uc: uc}
}

func (h *AdminConfirmDealInvoiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

// AdminExpireDealInvoiceHandler expires a payment-pending invoice past due (operator).
type AdminExpireDealInvoiceHandler struct {
	tx TxRunner
	uc *billingapp.ExpireDealInvoice
}

func NewAdminExpireDealInvoiceHandler(tx TxRunner, uc *billingapp.ExpireDealInvoice) *AdminExpireDealInvoiceHandler {
	return &AdminExpireDealInvoiceHandler{tx: tx, uc: uc}
}

func (h *AdminExpireDealInvoiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		if errors.Is(err, billingapp.ErrInvoiceNotExpired) || errors.Is(err, wallet.ErrInvoiceNotExpirable) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AdminMarkSellerPayoutReadyHandler transitions PENDING → READY.
type AdminMarkSellerPayoutReadyHandler struct {
	tx TxRunner
	uc *billingapp.MarkSellerPayoutReady
}

func NewAdminMarkSellerPayoutReadyHandler(tx TxRunner, uc *billingapp.MarkSellerPayoutReady) *AdminMarkSellerPayoutReadyHandler {
	return &AdminMarkSellerPayoutReadyHandler{tx: tx, uc: uc}
}

func (h *AdminMarkSellerPayoutReadyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payoutID := chi.URLParam(r, "payoutID")
	if payoutID == "" {
		http.Error(w, `{"error":"MISSING_PAYOUT_ID"}`, http.StatusBadRequest)
		return
	}
	var out *wallet.SellerPayout
	if err := h.tx.WithinTx(r.Context(), func(ctx context.Context) error {
		p, err := h.uc.Execute(ctx, payoutID)
		if err != nil {
			return err
		}
		out = p
		return nil
	}); err != nil {
		if errors.Is(err, billingapp.ErrSellerPayoutNotFound) {
			http.Error(w, `{"error":"PAYOUT_NOT_FOUND"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, wallet.ErrSellerPayoutWrongStatus) || errors.Is(err, wallet.ErrInvalidIdentifier) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sellerPayoutJSON(out))
}

// AdminMarkSellerPayoutPaidHandler credits the seller and marks payout PAID.
type AdminMarkSellerPayoutPaidHandler struct {
	tx TxRunner
	uc *billingapp.MarkSellerPayoutPaid
}

func NewAdminMarkSellerPayoutPaidHandler(tx TxRunner, uc *billingapp.MarkSellerPayoutPaid) *AdminMarkSellerPayoutPaidHandler {
	return &AdminMarkSellerPayoutPaidHandler{tx: tx, uc: uc}
}

func (h *AdminMarkSellerPayoutPaidHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payoutID := chi.URLParam(r, "payoutID")
	if payoutID == "" {
		http.Error(w, `{"error":"MISSING_PAYOUT_ID"}`, http.StatusBadRequest)
		return
	}
	var out *wallet.SellerPayout
	if err := h.tx.WithinTx(r.Context(), func(ctx context.Context) error {
		p, err := h.uc.Execute(ctx, payoutID)
		if err != nil {
			return err
		}
		out = p
		return nil
	}); err != nil {
		if errors.Is(err, billingapp.ErrSellerPayoutNotFound) {
			http.Error(w, `{"error":"PAYOUT_NOT_FOUND"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, wallet.ErrSellerPayoutWrongStatus) || errors.Is(err, wallet.ErrInvalidIdentifier) || errors.Is(err, wallet.ErrInvoiceCurrencyMismatch) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sellerPayoutJSON(out))
}
