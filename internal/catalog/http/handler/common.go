package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalog "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/http"
	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/go-chi/chi/v5"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeHTTPError(w http.ResponseWriter, err error) {
	he := httpapi.MapError(err)
	slog.Warn("catalog_http_error",
		"component", "http.handler",
		"bounded_context", "catalog",
		"status", he.Status,
		"code", he.Code,
		"message", he.Message,
		"error", err,
	)
	writeJSON(w, he.Status, httpapi.ErrorResponse{Code: he.Code, Message: he.Message})
}

func productIDFromRequest(r *http.Request) (string, error) {
	if id := chi.URLParam(r, "productID"); id != "" {
		return id, nil
	}
	return "", catalog.ErrInvalidIdentifier
}

func lotIDFromRequest(r *http.Request) (string, error) {
	if id := chi.URLParam(r, "lotID"); id != "" {
		return id, nil
	}
	return "", catalog.ErrInvalidIdentifier
}

func companyIDFromRequest(r *http.Request) string {
	if companyID, ok := identityauth.CompanyIDFromContext(r.Context()); ok {
		return companyID
	}
	return r.Header.Get("X-Company-ID")
}

// catalogActorFromIdentity maps identity JWT claims to catalog's Actor (catalog must not import identity in app/).
func catalogActorFromIdentity(ident identityauth.Identity) catalogapp.Actor {
	if identity.IncludesRole(ident.Role, identity.RoleAdmin) {
		return catalogapp.Actor{
			CompanyID:           ident.CompanyID,
			Kind:                catalogapp.ActorKindPlatformAdmin,
			SellerCatalogAccess: true,
		}
	}
	sellerSide := identity.IncludesRole(ident.Role, identity.RoleSeller)
	return catalogapp.Actor{
		CompanyID:           ident.CompanyID,
		Kind:                catalogapp.ActorKindCompany,
		SellerCatalogAccess: sellerSide,
	}
}

// catalogRequestContext attaches company id and catalog actor from JWT when present, else company header only.
func catalogRequestContext(r *http.Request) context.Context {
	base := r.Context()
	if ident, ok := identityauth.IdentityFromContext(base); ok {
		return catalogapp.WithActor(base, catalogActorFromIdentity(ident))
	}
	return catalogapp.WithCompanyID(base, companyIDFromRequest(r))
}
