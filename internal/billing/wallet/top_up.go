package wallet

import "time"

type TopUpStatus string

const (
	TopUpPending   TopUpStatus = "PENDING"
	TopUpSucceeded TopUpStatus = "SUCCEEDED"
	TopUpFailed    TopUpStatus = "FAILED"
	TopUpCancelled TopUpStatus = "CANCELLED"
)

type TopUp struct {
	ID                string
	CompanyID         string
	AccountID         string
	Amount            int64
	Currency          Currency
	Status            TopUpStatus
	Provider          string
	ProviderPaymentID string
	ConfirmationURL   string
	CreatedAt         time.Time
	ConfirmedAt       *time.Time
	FailedAt          *time.Time
}

func NewTopUp(id, companyID, accountID string, amount int64, currency Currency, provider string, now time.Time) (*TopUp, error) {
	if isBlank(id) || isBlank(companyID) || isBlank(accountID) || isBlank(provider) {
		return nil, ErrInvalidIdentifier
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if currency != CurrencyRUB {
		return nil, ErrUnsupportedCurrency
	}
	return &TopUp{
		ID:          id,
		CompanyID:   companyID,
		AccountID:   accountID,
		Amount:      amount,
		Currency:    currency,
		Status:      TopUpPending,
		Provider:    provider,
		CreatedAt:   now.UTC(),
		ConfirmedAt: nil,
		FailedAt:    nil,
	}, nil
}

func (t *TopUp) AttachProviderPayment(providerPaymentID, confirmationURL string) error {
	if t.Status != TopUpPending {
		return ErrInvalidTopUpStatus
	}
	if isBlank(providerPaymentID) {
		return ErrInvalidIdentifier
	}
	t.ProviderPaymentID = providerPaymentID
	t.ConfirmationURL = confirmationURL
	return nil
}

func (t *TopUp) MarkSucceeded(now time.Time) error {
	if t.Status != TopUpPending {
		return ErrInvalidTopUpStatus
	}
	if isBlank(t.ProviderPaymentID) {
		return ErrInvalidIdentifier
	}
	t.Status = TopUpSucceeded
	ts := now.UTC()
	t.ConfirmedAt = &ts
	return nil
}

func (t *TopUp) MarkFailed(now time.Time) error {
	if t.Status != TopUpPending {
		return ErrInvalidTopUpStatus
	}
	t.Status = TopUpFailed
	ts := now.UTC()
	t.FailedAt = &ts
	return nil
}
