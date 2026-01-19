// Auction invariants:
//
// 1. Bids can be placed only in PUBLISHED state
// 2. Auction can be closed only once
// 3. Winner is determined only at close time
// 4. Highest bid wins
// 5. Tie chooses first
// 6. Auction with no bids can be cancelled
// 7. No state transition backwards

// Инварианты аукциона:
//
// 1. Ставки можно делать только в состоянии PUBLISHED
// 2. Аукцион можно закрыть только один раз
// 3. Победитель определяется только в момент закрытия
// 4. Побеждает ставка с наибольшим значением
// 5. В случае ничьей выигрывает первая ставка
// 6. Аукцион без ставок можно отменить
// 7. Переход в исходное состояние невозможен

package auction

type Auction struct {
	ID     string
	state  State
	bids   []Bid
	winner *Bid
}

func NewAuction(id string) *Auction {
	return &Auction{
		ID:    id,
		state: StateDraft,
	}
}

func (a *Auction) Publish() error {
	return a.transitionTo(StatePublished)
}

func (a *Auction) PlaceBid(b Bid) error {
	if a.state != StatePublished {
		return ErrAuctionNotActive
	}
	a.bids = append(a.bids, b)
	return nil
}

func (a *Auction) Close() error {
	if a.state != StatePublished {
		return ErrCannotCloseAuction
	}
	winner, ok := determineWinner(a.bids)
	if !ok {
		return ErrNoBids
	}
	if err := a.transitionTo(StateClosed); err != nil {
		return err
	}
	a.winner = &winner
	return a.transitionTo(StateWon)
}

func (a *Auction) Cancel() error {
	if a.state != StatePublished {
		return ErrInvalidStateTransition
	}
	if len(a.bids) > 0 {
		return ErrCannotCancelWithBids
	}
	return a.transitionTo(StateCancelled)
}

func (a *Auction) State() State {
	return a.state
}

func (a *Auction) Bids() []Bid {
	if len(a.bids) == 0 {
		return nil
	}
	out := make([]Bid, len(a.bids))
	copy(out, a.bids)
	return out
}

func (a *Auction) Winner() (Bid, bool) {
	if a.winner == nil {
		return Bid{}, false
	}
	return *a.winner, true
}