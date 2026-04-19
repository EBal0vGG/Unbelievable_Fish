package handler

import (
	"net/http"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
)

type GetCurrentUserHandler struct {
	uc *identityapp.GetCurrentUser
}

func NewGetCurrentUserHandler(uc *identityapp.GetCurrentUser) *GetCurrentUserHandler {
	return &GetCurrentUserHandler{uc: uc}
}

func (h *GetCurrentUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	meta := readMeta(r)
	userID, ok := identityauth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, identityauth.ErrIdentityNotFound, meta)
		return
	}

	result, err := h.uc.Execute(r.Context(), identityapp.GetCurrentUserQuery{
		UserID: userID,
	})
	if err != nil {
		writeError(w, err, meta)
		return
	}

	writeJSON(w, http.StatusOK, httpapi.NewUserResponse(result))
}
