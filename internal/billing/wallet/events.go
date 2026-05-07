package wallet

// Domain events for future billing outbox (stage 2+).

type AccountCreated struct {
	AccountID string
	CompanyID string
	Currency  Currency
}

type BalanceToppedUp struct {
	AccountID string
	CompanyID string
	Amount    int64
	Currency  Currency
}

type AuctionDepositReserved struct {
	AuctionID string
	CompanyID string
	Amount    int64
	Currency  Currency
}

type AuctionDepositReleased struct {
	AuctionID string
	CompanyID string
	Amount    int64
	Currency  Currency
	Reason    string
}

type AuctionDepositCaptured struct {
	AuctionID string
	CompanyID string
	Amount    int64
	Currency  Currency
	Reason    string
}

type PlatformFeeCaptured struct {
	AuctionID string
	CompanyID string
	Amount    int64
	Currency  Currency
}

type PlatformFeePaymentRequired struct {
	AuctionID string
	CompanyID string
	AmountDue int64
	Currency  Currency
}
