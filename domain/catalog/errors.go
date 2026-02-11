package catalog

import "errors"

var (
	ErrInvalidIdentifier        = errors.New("invalid identifier")
	ErrInvalidEnum              = errors.New("invalid enum value")
	ErrInvalidQuantity          = errors.New("invalid quantity")
	ErrForbiddenStateTransition = errors.New("forbidden state transition")
	ErrPublishingRuleViolation  = errors.New("publishing rule violation")
	ErrModificationNotAllowed   = errors.New("modification not allowed")
	ErrInvalidPrice             = errors.New("invalid price")
	ErrInvalidSchedule          = errors.New("invalid schedule")
	ErrAlreadyAssigned          = errors.New("already assigned")
	ErrAuctionIDRequired        = errors.New("auction id required")
	ErrInvalidWeight            = errors.New("invalid weight")
)
