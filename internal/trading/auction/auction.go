// Auction invariants:
//
// 1. Bids can be placed only in PUBLISHED state
// 2. Auction can be closed only once
// 3. Winner is determined only at close time
// 4. Highest bid wins
// 5. Tie chooses first
// 6. Bid amount cannot be lower than current price
// 7. Auction with no bids can be cancelled
// 8. No state transition backwards
// 9. Bids before start or after end are rejected
// 10. Late bids extend the auction end time

// Инварианты аукциона:
//
// 1. Ставки можно делать только в состоянии PUBLISHED
// 2. Аукцион можно закрыть только один раз
// 3. Победитель определяется только в момент закрытия
// 4. Побеждает ставка с наибольшим значением
// 5. В случае ничьей выигрывает первая ставка
// 6. Ставка не может быть меньше текущей цены
// 7. Аукцион без ставок можно отменить
// 8. Переход в исходное состояние невозможен
// 9. Ставки до начала или после конца отклоняются
// 10. Поздние ставки продлевают конец аукциона

package auction

import "time"

type Auction struct {
	ID                string
	LotID             string
	state             State
	bids              []Bid
	winner            *Bid
	startsAt          time.Time
	endsAt            time.Time
	currentPrice      int64
	extensionWindow   time.Duration
	extensionDuration time.Duration
}

const (
	defaultExtensionWindow   = 5 * time.Minute
	defaultExtensionDuration = 5 * time.Minute
)

func NewAuction(id, lotID string, startsAt, endsAt time.Time) (*Auction, error) {
	if id == "" {
		return nil, ErrAuctionIDEmpty
	}
	if lotID == "" {
		return nil, ErrLotIDEmpty
	}
	if startsAt.IsZero() || endsAt.IsZero() || !startsAt.Before(endsAt) {
		return nil, ErrInvalidSchedule
	}
	return &Auction{
		ID:                id,
		LotID:             lotID,
		state:             StateDraft,
		startsAt:          startsAt,
		endsAt:            endsAt,
		extensionWindow:   defaultExtensionWindow,
		extensionDuration: defaultExtensionDuration,
	}, nil
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
	if b.PlacedAt().Before(a.startsAt) {
		return nil, ErrAuctionNotStarted
	}
	if b.PlacedAt().After(a.endsAt) {
		return nil, ErrAuctionAlreadyEnded
	}
	if b.Amount() < a.currentPrice {
		return nil, ErrBidTooLow
	}
	a.bids = append(a.bids, b)
	a.currentPrice = b.Amount()
	newEndAt := a.endsAt
	if a.endsAt.Sub(b.PlacedAt()) <= a.extensionWindow {
		newEndAt = a.endsAt.Add(a.extensionDuration)
		a.endsAt = newEndAt
	}
	return []Event{
		BidPlaced{
			AuctionID:       a.ID,
			BidderCompanyID: b.BidderCompanyID(),
			Amount:          b.Amount(),
			PlacedAt:        b.PlacedAt(),
			NewEndAt:        newEndAt,
		},
	}, nil
}

func (a *Auction) Close() ([]Event, error) {
	if a.state != StatePublished {
		return nil, ErrCannotCloseAuction
	}
	if len(a.bids) == 0 {
		if err := a.transitionTo(StateCancelled); err != nil {
			return nil, err
		}
		return []Event{
			AuctionCancelled{AuctionID: a.ID},
		}, nil
	}
	winner, _ := determineWinner(a.bids)
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

func (a *Auction) StartsAt() time.Time {
	return a.startsAt
}

func (a *Auction) EndsAt() time.Time {
	return a.endsAt
}

func (a *Auction) CurrentPrice() int64 {
	return a.currentPrice
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
