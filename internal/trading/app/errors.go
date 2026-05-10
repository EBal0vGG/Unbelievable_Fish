package app

import "errors"

var (
	ErrNilUnitOfWork             = errors.New("unit of work is nil")
	ErrNilAuctionIDFactory       = errors.New("auction id factory is nil")
	ErrNilAuctionQueryRepository = errors.New("auction query repository is nil")
	ErrNilDepositService         = errors.New("deposit service is nil")
	ErrNotFound                  = errors.New("auction not found")
	// ErrInsufficientFundsForDeposit is returned when the buyer cannot reserve the auction deposit (mapped to HTTP 409 in trading API).
	ErrInsufficientFundsForDeposit = errors.New("insufficient funds for auction deposit")
	// ErrCloseForbiddenBeforeEndWithBids is returned when a non-privileged actor tries to close while the auction has bids and end time is not reached.
	ErrCloseForbiddenBeforeEndWithBids = errors.New("cannot close auction with bids before scheduled end")
	// ErrCancelAuctionNotAllowed is returned when HTTP dispatch cannot route cancel (e.g. unsupported actor).
	ErrCancelAuctionNotAllowed = errors.New("cancel auction not allowed for this actor")
)
