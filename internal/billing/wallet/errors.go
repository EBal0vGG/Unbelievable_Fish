package wallet

import "errors"

var (
	ErrInvalidIdentifier   = errors.New("invalid identifier")
	ErrUnsupportedCurrency = errors.New("unsupported currency")
	ErrInvalidAmount       = errors.New("amount must be positive")
	ErrInsufficientFunds   = errors.New("insufficient available funds")
	ErrInsufficientHeld    = errors.New("insufficient held funds")
	ErrInvalidDepositState = errors.New("invalid deposit status for operation")
	ErrInvalidTopUpStatus  = errors.New("invalid top-up status for operation")
)
