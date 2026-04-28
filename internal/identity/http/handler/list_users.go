package handler

import (
	"net/http"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/http"
)

type ListUsersHandler struct {
	uc *identityapp.ListUsers
}

func NewListUsersHandler(uc *identityapp.ListUsers) *ListUsersHandler {
	return &ListUsersHandler{uc: uc}
}

func (h *ListUsersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	meta := readMeta(r)
	users, err := h.uc.Execute(r.Context())
	if err != nil {
		writeError(w, err, meta)
		return
	}
	resp := make([]httpapi.UserResponse, 0, len(users))
	for _, user := range users {
		resp = append(resp, httpapi.NewUserResponse(user))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": resp})
}
