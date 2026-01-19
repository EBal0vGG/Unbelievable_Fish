package auction

var allowedTransitions = map[State]map[State]struct{}{
	StateDraft: {
		StatePublished: {},
	},
	StatePublished: {
		StateClosed:    {},
		StateCancelled: {},
	},
	StateClosed: {
		StateWon: {},
	},
}

func (a *Auction) transitionTo(next State) error {
	if !canTransition(a.state, next) {
		return ErrInvalidStateTransition
	}
	a.state = next
	return nil
}

func canTransition(from, to State) bool {
	targets, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	_, ok = targets[to]
	return ok
}
