package app

import "context"

type companyIDContextKey struct{}

func WithCompanyID(ctx context.Context, companyID string) context.Context {
	return context.WithValue(ctx, companyIDContextKey{}, companyID)
}

func companyIDFromContext(ctx context.Context) (string, bool) {
	companyID, ok := ctx.Value(companyIDContextKey{}).(string)
	if !ok || companyID == "" {
		return "", false
	}
	return companyID, true
}
