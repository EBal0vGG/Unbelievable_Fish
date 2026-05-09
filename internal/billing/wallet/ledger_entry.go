package wallet

import "time"

type LedgerEntryType string

const (
	LedgerTopUpConfirmed      LedgerEntryType = "TOP_UP_CONFIRMED"
	LedgerBidDepositReserved  LedgerEntryType = "BID_DEPOSIT_RESERVED"
	LedgerBidDepositReleased  LedgerEntryType = "BID_DEPOSIT_RELEASED"
	LedgerBidDepositCaptured  LedgerEntryType = "BID_DEPOSIT_CAPTURED"
	LedgerPlatformFeeCaptured LedgerEntryType = "PLATFORM_FEE_CAPTURED"
	LedgerPlatformFeeDue      LedgerEntryType = "PLATFORM_FEE_DUE"
	// LedgerSellerPayoutCredited — зачисление на available продавца при переводе payout в PAID (Stage 13+).
	LedgerSellerPayoutCredited LedgerEntryType = "SELLER_PAYOUT_CREDITED"
)

type LedgerEntry struct {
	ID            string
	AccountID     string
	CompanyID     string
	Currency      Currency
	Amount        int64
	EntryType     LedgerEntryType
	ReferenceType string
	ReferenceID   string
	Reason        string
	CreatedAt     time.Time
}
