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

func (a *Auction) Publish() ([]Event, error) {
	if a.state != StateDraft {
		return nil, ErrAuctionCannotBePublished
	}
	if err := a.transitionTo(StatePublished); err != nil {
		return nil, err
	}
	return []Event{
		AuctionPublished{AuctionID: a.ID},
	}, nil
}

func (a *Auction) PlaceBid(b Bid) ([]Event, error) {
	if a.state != StatePublished {
		return nil, ErrAuctionNotActive
	}
	a.bids = append(a.bids, b)
	return []Event{
		BidPlaced{
			AuctionID:       a.ID,
			BidderCompanyID: b.BidderCompanyID(),
			Amount:          b.Amount(),
			PlacedAt:        b.PlacedAt(),
		},
	}, nil
}

func (a *Auction) Close() ([]Event, error) {
	if a.state != StatePublished {
		return nil, ErrCannotCloseAuction
	}
	winner, ok := determineWinner(a.bids)
	if !ok {
		return nil, ErrNoBids
	}
	if err := a.transitionTo(StateClosed); err != nil {
		return nil, err
	}
	a.winner = &winner
	if err := a.transitionTo(StateWon); err != nil {
		return nil, err
	}
	return []Event{
		AuctionClosed{AuctionID: a.ID},
		AuctionWon{
			AuctionID:       a.ID,
			WinnerCompanyID: winner.BidderCompanyID(),
			FinalPrice:      winner.Amount(),
		},
	}, nil
}

func (a *Auction) Cancel() ([]Event, error) {
	if a.state != StatePublished {
		return nil, ErrInvalidStateTransition
	}
	if len(a.bids) > 0 {
		return nil, ErrCannotCancelWithBids
	}
	if err := a.transitionTo(StateCancelled); err != nil {
		return nil, err
	}
	return []Event{
		AuctionCancelled{AuctionID: a.ID},
	}, nil
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
