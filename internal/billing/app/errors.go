package app

import "errors"

var (
	ErrInsufficientFundsForDeposit = errors.New("insufficient funds for auction deposit")
	ErrDepositNotHeld              = errors.New("auction deposit is not in HELD state")
	ErrDepositNotFound             = errors.New("auction deposit not found")
	ErrAccountNotFound             = errors.New("billing account not found")
)
