package catalog

import "errors"

var (
	ErrInvalidIdentifier       = errors.New("invalid identifier")
	ErrInvalidEnum             = errors.New("invalid enum value")
	ErrInvalidQuantity         = errors.New("invalid quantity")
	ErrForbiddenStateTransition = errors.New("forbidden state transition")
	ErrPublishingRuleViolation = errors.New("publishing rule violation")
	ErrModificationNotAllowed  = errors.New("modification not allowed")
)
