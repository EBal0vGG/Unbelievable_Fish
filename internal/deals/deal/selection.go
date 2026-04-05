package deal

import "time"

type WinnerSelectionStatus string

const (
	WinnerSelectionActive    WinnerSelectionStatus = "active"
	WinnerSelectionExhausted WinnerSelectionStatus = "exhausted"
)

// WinnerSelection tracks post-auction candidate progression.
type WinnerSelection struct {
	AuctionID    string
	Candidates   []string
	CurrentIndex int
	Status       WinnerSelectionStatus

	FinalPrice int64
	WonAt      time.Time

	SupplierID      string
	ProductSnapshot ProductSnapshot
	DealID          string
}

func NewWinnerSelection(
	auctionID string,
	candidates []string,
	finalPrice int64,
	wonAt time.Time,
	supplierID string,
	snapshot ProductSnapshot,
) *WinnerSelection {
	return &WinnerSelection{
		AuctionID:       auctionID,
		Candidates:      candidates,
		CurrentIndex:    0,
		Status:          WinnerSelectionActive,
		FinalPrice:      finalPrice,
		WonAt:           wonAt,
		SupplierID:      supplierID,
		ProductSnapshot: snapshot,
	}
}

func (s *WinnerSelection) CurrentCandidate() (string, bool) {
	if s == nil || s.Status == WinnerSelectionExhausted {
		return "", false
	}
	if s.CurrentIndex < 0 || s.CurrentIndex >= len(s.Candidates) {
		return "", false
	}
	if s.Candidates[s.CurrentIndex] == "" {
		return "", false
	}
	return s.Candidates[s.CurrentIndex], true
}

func (s *WinnerSelection) Advance() bool {
	if s == nil {
		return false
	}
	s.CurrentIndex++
	if s.CurrentIndex >= len(s.Candidates) {
		s.Status = WinnerSelectionExhausted
		return false
	}
	return true
}

func (s *WinnerSelection) MarkExhausted() {
	if s == nil {
		return
	}
	s.Status = WinnerSelectionExhausted
}
