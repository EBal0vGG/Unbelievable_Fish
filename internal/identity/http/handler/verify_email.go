package handler

import (
	"net/http"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
)

type VerifyEmailHandler struct {
	uc *identityapp.VerifyEmail
}

func NewVerifyEmailHandler(uc *identityapp.VerifyEmail) *VerifyEmailHandler {
	return &VerifyEmailHandler{uc: uc}
}

func (h *VerifyEmailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta := readMeta(r)
	token := r.URL.Query().Get("token")
	if r.Method == http.MethodPost {
		var req httpapi.VerifyEmailRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
			return
		}
		token = req.Token
	}

	result, err := h.uc.Execute(r.Context(), identityapp.VerifyEmailCommand{Token: token})
	if err != nil {
		writeError(w, err, meta)
		return
	}

	writeJSON(w, http.StatusOK, httpapi.VerifyEmailResponse{
		Status:          "verified",
		AlreadyVerified: result.AlreadyVerified,
	})
}

type ResendVerificationHandler struct {
	uc *identityapp.ResendVerification
}

func NewResendVerificationHandler(uc *identityapp.ResendVerification) *ResendVerificationHandler {
	return &ResendVerificationHandler{uc: uc}
}

func (h *ResendVerificationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta := readMeta(r)
	var req httpapi.ResendVerificationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}

	result, err := h.uc.Execute(r.Context(), identityapp.ResendVerificationCommand{Login: req.Login})
	if err != nil {
		writeError(w, err, meta)
		return
	}

	status := "sent"
	if result.AlreadyVerified {
		status = "already_verified"
	}
	writeJSON(w, http.StatusOK, httpapi.ResendVerificationResponse{
		Status:          status,
		AlreadyVerified: result.AlreadyVerified,
	})
}
