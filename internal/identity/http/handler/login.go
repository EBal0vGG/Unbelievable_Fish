package handler

import (
	"net/http"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
)

type LoginHandler struct {
	uc *identityapp.Login
}

func NewLoginHandler(uc *identityapp.Login) *LoginHandler {
	return &LoginHandler{uc: uc}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta := readMeta(r)
	var req httpapi.LoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}

	result, err := h.uc.Execute(r.Context(), identityapp.LoginCommand{
		Login:    req.Login,
		Password: req.Password,
	})
	if err != nil {
		writeError(w, err, meta)
		return
	}

	writeJSON(w, http.StatusOK, httpapi.NewLoginResponse(result))
}
