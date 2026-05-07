package auction

import "time"

func RehydrateAuction(
	id string,
	lotID string,
	state State,
	startsAt time.Time,
	endsAt time.Time,
	startPrice int64,
	currentPrice int64,
	minBidStep int64,
	leaderCompanyID string,
) (*Auction, error) {
	if id == "" {
		return nil, ErrAuctionIDEmpty
	}
	if lotID == "" {
		return nil, ErrLotIDEmpty
	}
	if startsAt.IsZero() || endsAt.IsZero() || !startsAt.Before(endsAt) {
		return nil, ErrInvalidSchedule
	}
	if !isValidState(state) {
		return nil, ErrInvalidStateTransition
	}
	if minBidStep <= 0 {
		return nil, ErrInvalidMinBidStep
	}
	if startPrice < 0 {
		return nil, ErrInvalidStartPrice
	}
	return &Auction{
		ID:                id,
		LotID:             lotID,
		state:             state,
		startsAt:          startsAt,
		endsAt:            endsAt,
		startPrice:        startPrice,
		currentPrice:      currentPrice,
		minBidStep:        minBidStep,
		leaderCompanyID:   leaderCompanyID,
		extensionWindow:   defaultExtensionWindow,
		extensionDuration: defaultExtensionDuration,
	}, nil
}

func isValidState(state State) bool {
	switch state {
	case StateDraft, StatePublished, StateClosed, StateWon, StateCancelled:
		return true
	default:
		return false
	}
}
