package auction

import "time"

func RehydrateAuction(
	id string,
	lotID string,
	state State,
	startsAt time.Time,
	endsAt time.Time,
	currentPrice int64,
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
	return &Auction{
		ID:                id,
		LotID:             lotID,
		state:             state,
		startsAt:          startsAt,
		endsAt:            endsAt,
		currentPrice:      currentPrice,
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
