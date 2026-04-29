package handler

import (
	"net/http"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
	"github.com/go-chi/chi/v5"
)

type PromoteUserAdminHandler struct {
	uc *identityapp.PromoteUserToAdmin
}

func NewPromoteUserAdminHandler(uc *identityapp.PromoteUserToAdmin) *PromoteUserAdminHandler {
	return &PromoteUserAdminHandler{uc: uc}
}

func (h *PromoteUserAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta := readMeta(r)
	userID, err := userIDFromPromoteRequest(r)
	if err != nil {
		writeError(w, err, meta)
		return
	}
	user, err := h.uc.Execute(r.Context(), identityapp.PromoteUserToAdminCommand{UserID: userID})
	if err != nil {
		writeError(w, err, meta)
		return
	}
	writeJSON(w, http.StatusAccepted, httpapi.NewUserResponse(user))
}

func userIDFromPromoteRequest(r *http.Request) (string, error) {
	if userID := chi.URLParam(r, "userID"); userID != "" {
		return userID, nil
	}
	return "", httpapi.ErrInvalidPath
}
