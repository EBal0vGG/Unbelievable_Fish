package app

import "errors"

var (
	ErrNilUnitOfWork      = errors.New("unit of work is nil")
	ErrNilAuctionIDFactory = errors.New("auction id factory is nil")
	ErrNotFound            = errors.New("auction not found")
)
