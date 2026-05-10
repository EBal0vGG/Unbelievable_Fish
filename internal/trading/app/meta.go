package app

import "context"

type CommandMeta struct {
	CompanyID     string
	UserID        string
	ActorKind     ActorKind
	CorrelationID string
	CausationID   string
}

type commandMetaKey struct{}

func WithCommandMeta(ctx context.Context, meta CommandMeta) context.Context {
	return context.WithValue(ctx, commandMetaKey{}, meta)
}

func CommandMetaFromContext(ctx context.Context) (CommandMeta, bool) {
	meta, ok := ctx.Value(commandMetaKey{}).(CommandMeta)
	return meta, ok
}
