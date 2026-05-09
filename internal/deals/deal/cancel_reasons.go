package deal

// Reasons for cancelling a deal. Use these constants from HTTP and schedulers so integration can rely on stable strings.
const (
	DealCancelReasonWinnerRejected          = "WINNER_REJECTED"
	DealCancelReasonConfirmationTimeout     = "CONFIRMATION_TIMEOUT"
	DealCancelReasonLegacyDeadlineExceeded  = "deadline exceeded"
	// DealCancelReasonPaymentTimeout — просрочка оплаты инвойса после подтверждения сделки (отдельная ветка Cancel, не ReasonForfeitsWinnerDeposit).
	DealCancelReasonPaymentTimeout = "PAYMENT_TIMEOUT"
)

// ReasonForfeitsWinnerDeposit reports whether the buyer's auction deposit should be captured (integration → billing).
func ReasonForfeitsWinnerDeposit(reason string) bool {
	switch reason {
	case DealCancelReasonWinnerRejected, DealCancelReasonConfirmationTimeout, DealCancelReasonLegacyDeadlineExceeded:
		return true
	default:
		return false
	}
}

// NormalizeWinnerForfeitReason maps legacy reasons to the canonical value stored on WinnerRejected / ledger.
func NormalizeWinnerForfeitReason(reason string) string {
	if reason == DealCancelReasonLegacyDeadlineExceeded {
		return DealCancelReasonConfirmationTimeout
	}
	return reason
}
