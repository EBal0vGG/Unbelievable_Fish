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
)
