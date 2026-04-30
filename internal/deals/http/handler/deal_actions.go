package handler

import (
	"net/http"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
)

type RequestDealConfirmationHandler struct{ uc *app.RequestDealConfirmation }

func NewRequestDealConfirmationHandler(uc *app.RequestDealConfirmation) *RequestDealConfirmationHandler {
	return &RequestDealConfirmationHandler{uc: uc}
}

func (h *RequestDealConfirmationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.RequestDealConfirmationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}

	confirmation, err := h.uc.Execute(r.Context(), meta, dealID, app.RequestDealConfirmationCommand{
		Stage:                 deal.DealConfirmationStage(req.Stage),
		VerificationMethod:    deal.VerificationMethod(req.VerificationMethod),
		VerificationTokenHash: req.VerificationTokenHash,
		SignatureRef:          req.SignatureRef,
		Comment:               req.Comment,
		ExpiresAt:             req.ExpiresAt,
	})
	if err != nil {
		writeError(w, err, meta)
		return
	}
	writeCreatedJSON(w, httpapi.NewDealConfirmationResponse(confirmation))
}

type ApproveDealConfirmationHandler struct{ uc *app.ApproveDealConfirmation }

func NewApproveDealConfirmationHandler(uc *app.ApproveDealConfirmation) *ApproveDealConfirmationHandler {
	return &ApproveDealConfirmationHandler{uc: uc}
}

func (h *ApproveDealConfirmationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, confirmationID, err := readConfirmationIDsFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	confirmation, err := h.uc.Execute(r.Context(), meta, dealID, confirmationID)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	writeJSON(w, http.StatusOK, httpapi.NewDealConfirmationResponse(confirmation))
}

type RejectDealConfirmationHandler struct{ uc *app.RejectDealConfirmation }

func NewRejectDealConfirmationHandler(uc *app.RejectDealConfirmation) *RejectDealConfirmationHandler {
	return &RejectDealConfirmationHandler{uc: uc}
}

func (h *RejectDealConfirmationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, confirmationID, err := readConfirmationIDsFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.RejectDealConfirmationRequest
	if err := decodeJSON(r, &req); err != nil && r.ContentLength != 0 {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}
	confirmation, err := h.uc.Execute(r.Context(), meta, dealID, confirmationID, req.Reason)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	writeJSON(w, http.StatusOK, httpapi.NewDealConfirmationResponse(confirmation))
}

type ConfirmDealHandler struct{ uc *app.ConfirmDeal }

func NewConfirmDealHandler(uc *app.ConfirmDeal) *ConfirmDealHandler {
	return &ConfirmDealHandler{uc: uc}
}

func (h *ConfirmDealHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, dealID); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}

type PrepareContractHandler struct{ uc *app.PrepareContract }

func NewPrepareContractHandler(uc *app.PrepareContract) *PrepareContractHandler {
	return &PrepareContractHandler{uc: uc}
}

func (h *PrepareContractHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.PrepareContractRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, dealID, req.ContractNumber, req.DocumentURL); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}

type SignContractHandler struct{ uc *app.SignContract }

func NewSignContractHandler(uc *app.SignContract) *SignContractHandler {
	return &SignContractHandler{uc: uc}
}

func (h *SignContractHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.SignContractRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, dealID, req.SignatureRef); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}

type RequestPaymentHandler struct{ uc *app.RequestPayment }

func NewRequestPaymentHandler(uc *app.RequestPayment) *RequestPaymentHandler {
	return &RequestPaymentHandler{uc: uc}
}

func (h *RequestPaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.RequestPaymentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, dealID, req.InvoiceNumber, req.DueDate); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}

type MarkDealPaidHandler struct{ uc *app.MarkDealPaid }

func NewMarkDealPaidHandler(uc *app.MarkDealPaid) *MarkDealPaidHandler {
	return &MarkDealPaidHandler{uc: uc}
}

func (h *MarkDealPaidHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.MarkDealPaidRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, dealID, req.PaymentID, req.PaymentType); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}

type RequestShipmentHandler struct{ uc *app.RequestShipment }

func NewRequestShipmentHandler(uc *app.RequestShipment) *RequestShipmentHandler {
	return &RequestShipmentHandler{uc: uc}
}

func (h *RequestShipmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, dealID); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}

type MarkDealShippedHandler struct{ uc *app.MarkDealShipped }

func NewMarkDealShippedHandler(uc *app.MarkDealShipped) *MarkDealShippedHandler {
	return &MarkDealShippedHandler{uc: uc}
}

func (h *MarkDealShippedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.MarkDealShippedRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, dealID, req.TrackingNumber, req.Carrier); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}

type CompleteDealHandler struct{ uc *app.CompleteDeal }

func NewCompleteDealHandler(uc *app.CompleteDeal) *CompleteDealHandler {
	return &CompleteDealHandler{uc: uc}
}

func (h *CompleteDealHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, dealID); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}

type CancelDealHandler struct{ uc *app.CancelDeal }

func NewCancelDealHandler(uc *app.CancelDeal) *CancelDealHandler { return &CancelDealHandler{uc: uc} }

func (h *CancelDealHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.CancelDealRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, dealID, req.Reason); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}

type UpdateDealPriceHandler struct{ uc *app.UpdateDealPrice }

func NewUpdateDealPriceHandler(uc *app.UpdateDealPrice) *UpdateDealPriceHandler {
	return &UpdateDealPriceHandler{uc: uc}
}

func (h *UpdateDealPriceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	dealID, err := readDealIDFromRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	var req httpapi.UpdateDealPriceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}
	if err := h.uc.Execute(r.Context(), meta, dealID, req.NewPrice); err != nil {
		writeError(w, err, meta)
		return
	}
	writeAccepted(w)
}
