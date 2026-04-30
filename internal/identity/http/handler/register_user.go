package handler

import (
	"net/http"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
)

type RegisterUserHandler struct {
	uc *identityapp.RegisterUser
}

func NewRegisterUserHandler(uc *identityapp.RegisterUser) *RegisterUserHandler {
	return &RegisterUserHandler{uc: uc}
}

func (h *RegisterUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta := readMeta(r)
	var req httpapi.RegisterUserRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}

	result, err := h.uc.Execute(r.Context(), identityapp.RegisterUserCommand{
		CompanyID:     req.CompanyID,
		CompanyINN:    req.CompanyINN,
		CompanyOGRN:   req.CompanyOGRN,
		Name:          req.Name,
		Role:          req.Role,
		Login:         req.Login,
		Password:      req.Password,
		AcceptedTerms: req.AcceptedTerms,
		TermsVersion:  req.TermsVersion,
	})
	if err != nil {
		writeError(w, err, meta)
		return
	}

	writeJSON(w, http.StatusAccepted, httpapi.NewUserResponse(result))
}
