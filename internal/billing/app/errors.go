package app

import "errors"

var (
	ErrInsufficientFundsForDeposit = errors.New("insufficient funds for auction deposit")
	ErrDepositNotHeld              = errors.New("auction deposit is not in HELD state")
	ErrDepositNotFound             = errors.New("auction deposit not found")
	ErrAccountNotFound             = errors.New("billing account not found")
	ErrTopUpNotFound               = errors.New("billing top-up not found")
	ErrTopUpAmountMismatch         = errors.New("top-up amount or currency does not match payment details")
	ErrDealInvoiceNotFound         = errors.New("billing deal invoice not found")
)
