package app

// ActorKind classifies the catalog caller for authorization inside the catalog bounded context.
// It is not the identity service role enum: HTTP/cmd map JWT or job context to these values.
type ActorKind string

const (
	// ActorKindCompany is a tenant-scoped principal (buyer, seller, etc. distinguished by flags).
	ActorKindCompany ActorKind = "company"
	// ActorKindPlatformAdmin may list all products/lots and bypass ownership on mutations.
	ActorKindPlatformAdmin ActorKind = "platform_admin"
)

// Actor is the catalog application's view of who is acting (set at HTTP or integration boundary).
type Actor struct {
	CompanyID string
	Kind      ActorKind
	// SellerCatalogAccess: when Kind is ActorKindCompany, true if this principal may manage
	// seller catalog entities for CompanyID (create/publish products and lots, scoped list).
	SellerCatalogAccess bool
}

func (a Actor) isPlatformAdmin() bool {
	return a.Kind == ActorKindPlatformAdmin
}
