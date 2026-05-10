package app

// ActorKind classifies who initiated a trading command for policy checks in the application layer.
// It is intentionally not the identity service role enum: HTTP and workers map their auth context
// to these values before invoking use cases.
type ActorKind string

const (
	// ActorKindCompany is a normal company-scoped actor (buyer/seller distinction is enforced at HTTP).
	ActorKindCompany ActorKind = "company"
	// ActorKindPlatformAdmin is an operator allowed to force-close before end and similar policies.
	ActorKindPlatformAdmin ActorKind = "platform_admin"
	// ActorKindSystem is an internal automated actor (scheduler, integration jobs).
	ActorKindSystem ActorKind = "system"
)

func effectiveActorKind(meta CommandMeta) ActorKind {
	if meta.ActorKind != "" {
		return meta.ActorKind
	}
	return ActorKindCompany
}
