package auction

type State string

const (
	StateDraft     State = "DRAFT"
	StatePublished State = "PUBLISHED"
	StateClosed    State = "CLOSED"
	StateWon       State = "WON"
	StateCancelled State = "CANCELLED"
)