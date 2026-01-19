// Auction invariants:
//
// 1. Bids can be placed only in PUBLISHED state
// 2. Auction can be closed only once
// 3. Winner is determined only at close time
// 4. Highest bid wins
// 5. Auction with no bids can be cancelled
// 6. No state transition backwards

// Инварианты аукциона:
//
// 1. Ставки можно делать только в состоянии PUBLISHED
// 2. Аукцион можно закрыть только один раз
// 3. Победитель определяется только в момент закрытия
// 4. Побеждает ставка, набравшая наибольшее количество голосов
// 5. Аукцион без ставок можно отменить
// 6. Переход в исходное состояние невозможен

package auction

import "errors"

type Auction struct {
	ID     string
	state  State
	bids   []Bid
}

func NewAuction(id string) *Auction {
	return &Auction{
		ID:    id,
		state: StateDraft,
	}
}

func (a *Auction) Publish() error {
	if a.state != StateDraft {
		return errors.New("auction cannot be published")
	}
	a.state = StatePublished
	return nil
}

func (a *Auction) PlaceBid(b Bid) error {
	if a.state != StatePublished {
		return errors.New("auction not active")
	}
	a.bids = append(a.bids, b)
	return nil
}

func (a *Auction) Close() error {
	if a.state != StatePublished {
		return errors.New("cannot close auction")
	}
	a.state = StateClosed
	return nil
}