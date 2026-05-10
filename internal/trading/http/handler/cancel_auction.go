package handler

import (
	"net/http"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type CancelAuctionHandler struct {
	sellerUC *app.SellerCancelAuction
	adminUC  *app.AdminCancelAuction
}

func NewCancelAuctionHandler(sellerUC *app.SellerCancelAuction, adminUC *app.AdminCancelAuction) *CancelAuctionHandler {
	return &CancelAuctionHandler{sellerUC: sellerUC, adminUC: adminUC}
}

func (h *CancelAuctionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	meta, err := readCommandMeta(r)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	auctionID, err := readAuctionIDFromRequest(r)
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	ident, ok := identityauth.IdentityFromContext(r.Context())
	if !ok {
		handleCommandError(w, app.ErrCancelAuctionNotAllowed, meta)
		return
	}
	switch {
	case identity.IncludesRole(ident.Role, identity.RoleAdmin):
		err = h.adminUC.Execute(r.Context(), meta, auctionID)
	case identity.IncludesRole(ident.Role, identity.RoleSeller):
		err = h.sellerUC.Execute(r.Context(), meta, auctionID)
	default:
		err = app.ErrCancelAuctionNotAllowed
	}
	if err != nil {
		handleCommandError(w, err, meta)
		return
	}
	writeAccepted(w)
}
