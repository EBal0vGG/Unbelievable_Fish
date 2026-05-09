package app

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInsufficientFundsForDeposit = errors.New("insufficient funds for auction deposit")
	ErrDepositNotHeld              = errors.New("auction deposit is not in HELD state")
	ErrDepositNotFound             = errors.New("auction deposit not found")
	ErrAccountNotFound             = errors.New("billing account not found")
	ErrTopUpNotFound               = errors.New("billing top-up not found")
	ErrTopUpAmountMismatch         = errors.New("top-up amount or currency does not match payment details")
	ErrDealInvoiceNotFound         = errors.New("billing deal invoice not found")
	ErrSellerPayoutNotFound        = errors.New("billing seller payout not found")
	ErrSellerPayoutInvoiceMismatch = errors.New("billing: paid invoice fields do not match winner selection finalized payload")
)

// IsPostgresUniqueViolation reports PostgreSQL error 23505 (duplicate key).
// Used for idempotent Create under concurrent outbox/replay consumers.
func IsPostgresUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
