package app

import "errors"

var (
	ErrNotFound               = errors.New("not found")
	ErrFishNotFound           = errors.New("fish not found")
	ErrUnitNotFound           = errors.New("unit not found")
	ErrProcessingTypeNotFound = errors.New("processing type not found")
	ErrMissingCompanyID       = errors.New("missing company id")
)
