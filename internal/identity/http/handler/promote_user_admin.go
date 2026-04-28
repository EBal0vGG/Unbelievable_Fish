package handler

import (
	"net/http"
	"strings"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
)

type PromoteUserAdminHandler struct {
	uc *identityapp.PromoteUserToAdmin
}

func NewPromoteUserAdminHandler(uc *identityapp.PromoteUserToAdmin) *PromoteUserAdminHandler {
	return &PromoteUserAdminHandler{uc: uc}
}

func (h *PromoteUserAdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	meta := readMeta(r)
	userID, err := userIDFromPromotePath(r.URL.Path)
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

func userIDFromPromotePath(path string) (string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "users" || parts[2] != "promote-admin" || parts[1] == "" {
		return "", httpapi.ErrInvalidPath
	}
	return parts[1], nil
}
