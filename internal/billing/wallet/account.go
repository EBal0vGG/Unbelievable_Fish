package wallet

func isBlank(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

type Account struct {
	id        string
	companyID string
	currency  Currency

	available int64
	held      int64
}

func NewAccount(id, companyID string, currency Currency) (*Account, error) {
	if isBlank(id) || isBlank(companyID) {
		return nil, ErrInvalidIdentifier
	}
	if currency != CurrencyRUB {
		return nil, ErrUnsupportedCurrency
	}
	return &Account{
		id:        id,
		companyID: companyID,
		currency:  currency,
		available: 0,
		held:      0,
	}, nil
}

func RehydrateAccount(id, companyID string, currency Currency, available, held int64) (*Account, error) {
	a, err := NewAccount(id, companyID, currency)
	if err != nil {
		return nil, err
	}
	if available < 0 || held < 0 {
		return nil, ErrInvalidAmount
	}
	a.available = available
	a.held = held
	return a, nil
}

func (a *Account) ID() string            { return a.id }
func (a *Account) CompanyID() string     { return a.companyID }
func (a *Account) Currency() Currency    { return a.currency }
func (a *Account) Available() int64      { return a.available }
func (a *Account) Held() int64           { return a.held }
func (a *Account) Total() int64          { return a.available + a.held }

func (a *Account) Deposit(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	a.available += amount
	return nil
}

func (a *Account) Reserve(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.available < amount {
		return ErrInsufficientFunds
	}
	a.available -= amount
	a.held += amount
	return nil
}

func (a *Account) Release(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.held < amount {
		return ErrInsufficientHeld
	}
	a.held -= amount
	a.available += amount
	return nil
}

// Capture removes amount from held without returning it to available (e.g. platform fee or penalty).
func (a *Account) Capture(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.held < amount {
		return ErrInsufficientHeld
	}
	a.held -= amount
	return nil
}
