package wallet

import "time"

type DepositStatus string

const (
	DepositHeld     DepositStatus = "HELD"
	DepositReleased DepositStatus = "RELEASED"
	DepositCaptured DepositStatus = "CAPTURED"
	DepositSettled  DepositStatus = "SETTLED"
)

type AuctionDeposit struct {
	AuctionID string
	CompanyID string
	AccountID string
	Amount    int64
	Currency  Currency
	Status    DepositStatus

	CreatedAt  time.Time
	ReleasedAt *time.Time
	CapturedAt *time.Time
}

func NewAuctionDeposit(auctionID, companyID, accountID string, amount int64, currency Currency, now time.Time) (*AuctionDeposit, error) {
	if isBlank(auctionID) || isBlank(companyID) || isBlank(accountID) {
		return nil, ErrInvalidIdentifier
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if currency != CurrencyRUB {
		return nil, ErrUnsupportedCurrency
	}
	return &AuctionDeposit{
		AuctionID: auctionID,
		CompanyID: companyID,
		AccountID: accountID,
		Amount:    amount,
		Currency:  currency,
		Status:    DepositHeld,
		CreatedAt: now,
	}, nil
}

func (d *AuctionDeposit) MarkReleased(now time.Time) {
	d.Status = DepositReleased
	d.ReleasedAt = &now
}

func (d *AuctionDeposit) MarkCaptured(now time.Time) {
	d.Status = DepositCaptured
	d.CapturedAt = &now
}

func (d *AuctionDeposit) MarkSettled(now time.Time) {
	d.Status = DepositSettled
	d.ReleasedAt = &now
	d.CapturedAt = &now
}
