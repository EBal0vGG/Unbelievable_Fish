package handler

import (
	"net/http"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
)

type RegisterCompanyHandler struct {
	uc *identityapp.RegisterCompany
}

func NewRegisterCompanyHandler(uc *identityapp.RegisterCompany) *RegisterCompanyHandler {
	return &RegisterCompanyHandler{uc: uc}
}

func (h *RegisterCompanyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	meta := readMeta(r)
	var req httpapi.RegisterCompanyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, httpapi.BadRequest("INVALID_BODY", "invalid request body"), meta)
		return
	}

	result, err := h.uc.Execute(r.Context(), identityapp.RegisterCompanyCommand{
		Name: req.Name,
		INN:  req.INN,
		OGRN: req.OGRN,
	})
	if err != nil {
		writeError(w, err, meta)
		return
	}

	writeJSON(w, http.StatusAccepted, httpapi.NewCompanyResponse(result))
}
