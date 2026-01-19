// Bid invariants:
//
// 1. Company ID is no empty
// 2. Bid amound must be positive
// 3. Bid time is correct
//
// 1. Непустой ID компании
// 2. Положительное значение ставки
// 3. Корректное время ставки

package auction

import "time"

type Bid struct {
	bidderCompanyID string
	amount          int64 // rubles
	placedAt        time.Time
}

func NewBid(bidderCompanyID string, amount int64, placedAt time.Time) (Bid, error) {
	if bidderCompanyID == "" {
		return Bid{}, ErrBidderCompanyIDEmpty
	}
	if amount <= 0 {
		return Bid{}, ErrBidAmountNonPositive
	}
	if placedAt.IsZero() {
		return Bid{}, ErrBidPlacedAtZero
	}
	return Bid{
		bidderCompanyID: bidderCompanyID,
		amount:          amount,
		placedAt:        placedAt,
	}, nil
}

func (b Bid) BidderCompanyID() string {
	return b.bidderCompanyID
}

func (b Bid) Amount() int64 {
	return b.amount
}

func (b Bid) PlacedAt() time.Time {
	return b.placedAt
}
