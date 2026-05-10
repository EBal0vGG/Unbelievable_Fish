package app

import "context"

type actorContextKey struct{}

func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, a)
}

// WithCompanyID sets the acting company; it preserves existing Kind and SellerCatalogAccess if Actor was already set.
func WithCompanyID(ctx context.Context, companyID string) context.Context {
	prev, _ := ActorFromContext(ctx)
	prev.CompanyID = companyID
	return context.WithValue(ctx, actorContextKey{}, prev)
}

func ActorFromContext(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(actorContextKey{}).(Actor)
	if !ok || a.CompanyID == "" {
		return Actor{}, false
	}
	return a, true
}
