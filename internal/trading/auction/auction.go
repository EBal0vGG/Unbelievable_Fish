// Auction invariants:
//
// 1. Bids can be placed only in PUBLISHED state
// 2. Auction can be closed only once
// 3. Winner is determined only at close time
// 4. Highest bid wins
// 5. Tie chooses first
// 6. Bid amount must be strictly greater than current price
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
// 6. Ставка должна быть строго выше текущей цены
// 7. Аукцион без ставок можно отменить
// 8. Переход в исходное состояние невозможен
// 9. Ставки до начала или после конца отклоняются
// 10. Поздние ставки продлевают конец аукциона

package auction

import "time"

type Auction struct {
	ID    string
	LotID string

	state State

	startsAt time.Time
	endsAt   time.Time

	startPrice   int64
	currentPrice int64
	minBidStep   int64

	leaderCompanyID string

	extensionWindow   time.Duration
	extensionDuration time.Duration
}

const (
	defaultExtensionWindow   = 5 * time.Minute
	defaultExtensionDuration = 5 * time.Minute
	defaultMinBidStep        = int64(1)
)

func NewAuction(id, lotID string, startsAt, endsAt time.Time) (*Auction, error) {
	return NewAuctionWithPricing(id, lotID, startsAt, endsAt, 0, defaultMinBidStep)
}

func NewAuctionWithPricing(id, lotID string, startsAt, endsAt time.Time, startPrice, minBidStep int64) (*Auction, error) {
	if id == "" {
		return nil, ErrAuctionIDEmpty
	}
	if lotID == "" {
		return nil, ErrLotIDEmpty
	}
	if startsAt.IsZero() || endsAt.IsZero() || !startsAt.Before(endsAt) {
		return nil, ErrInvalidSchedule
	}
	if startPrice < 0 {
		return nil, ErrInvalidStartPrice
	}
	if minBidStep <= 0 {
		return nil, ErrInvalidMinBidStep
	}
	return &Auction{
		ID:                id,
		LotID:             lotID,
		state:             StateDraft,
		startsAt:          startsAt,
		endsAt:            endsAt,
		startPrice:        startPrice,
		currentPrice:      startPrice,
		minBidStep:        minBidStep,
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
	minAllowed := a.currentPrice
	if a.leaderCompanyID != "" {
		minAllowed = a.currentPrice + a.minBidStep
	}
	if b.Amount() < minAllowed {
		return nil, ErrBidStepTooSmall
	}
	a.currentPrice = b.Amount()
	a.leaderCompanyID = b.BidderCompanyID()
	newEndAt := a.endsAt
	if a.endsAt.Sub(b.PlacedAt()) <= a.extensionWindow {
		newEndAt = b.PlacedAt().Add(a.extensionDuration)
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

func (a *Auction) Close(bids []Bid) ([]Event, error) {
	if a.state != StatePublished {
		return nil, ErrCannotCloseAuction
	}
	if len(bids) == 0 {
		if err := a.transitionTo(StateCancelled); err != nil {
			return nil, err
		}
		return []Event{
			AuctionCancelled{AuctionID: a.ID},
		}, nil
	}
	winner, _ := determineWinner(bids)
	if err := a.transitionTo(StateClosed); err != nil {
		return nil, err
	}
	a.currentPrice = winner.Amount()
	a.leaderCompanyID = winner.BidderCompanyID()
	if err := a.transitionTo(StateWon); err != nil {
		return nil, err
	}
	return []Event{
		AuctionClosed{AuctionID: a.ID},
		AuctionWon{
			AuctionID:       a.ID,
			LotID:           a.LotID,
			WinnerCompanyID: collectWinnerCompanyIDs(bids),
			FinalPrice:      a.currentPrice,
		},
	}, nil
}

func (a *Auction) Cancel() ([]Event, error) {
	if a.state != StatePublished {
		return nil, ErrInvalidStateTransition
	}
	if a.currentPrice > 0 {
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

func (a *Auction) StartPrice() int64 {
	return a.startPrice
}

func (a *Auction) MinBidStep() int64 {
	return a.minBidStep
}

func (a *Auction) LeaderCompanyID() string {
	return a.leaderCompanyID
}

// maxWinnerCandidatesInAuctionWon matches CloseAuction winner persistence (top bid rows).
const maxWinnerCandidatesInAuctionWon = 3

func collectWinnerCompanyIDs(bids []Bid) []string {
	if len(bids) == 0 {
		return nil
	}
	n := maxWinnerCandidatesInAuctionWon
	if n > len(bids) {
		n = len(bids)
	}
	seen := make(map[string]struct{}, n)
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := bids[i].BidderCompanyID()
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (a *Auction) Winner() (string, int64, bool) {
	if a.leaderCompanyID == "" {
		return "", 0, false
	}
	return a.leaderCompanyID, a.currentPrice, true
}
