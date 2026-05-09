package deal

import "time"

type WinnerSelectionStatus string

const (
	WinnerSelectionActive                  WinnerSelectionStatus = "active"
	WinnerSelectionConfirmedPendingPayment WinnerSelectionStatus = "confirmed_pending_payment"
	WinnerSelectionFinalized               WinnerSelectionStatus = "finalized"
	WinnerSelectionExhausted             WinnerSelectionStatus = "exhausted"
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
	// DealID is the authoritative pointer to the current active deal attempt for this auction (updated on each fallback).
	// MarkCurrentConfirmed assigns DealID when it is still empty; otherwise it must match the deal being confirmed.
	// When Status is WinnerSelectionExhausted, DealID is cleared.
	DealID string
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
	if s == nil || s.Status != WinnerSelectionActive {
		return false
	}
	next := s.CurrentIndex + 1
	if next >= len(s.Candidates) {
		s.Status = WinnerSelectionExhausted
		return false
	}
	s.CurrentIndex = next
	return true
}

func (s *WinnerSelection) MarkCurrentConfirmed(dealID string) error {
	if s == nil {
		return ErrSelectionNotFound
	}
	if dealID == "" {
		return ErrDealIDRequired
	}
	if s.Status != WinnerSelectionActive {
		return ErrWinnerSelectionNotActive
	}
	if s.DealID == "" {
		s.DealID = dealID
	} else if s.DealID != dealID {
		return ErrStaleWinnerSelection
	}
	if _, ok := s.CurrentCandidate(); !ok {
		return ErrNoAvailableWinnerCandidate
	}
	s.Status = WinnerSelectionConfirmedPendingPayment
	return nil
}

func (s *WinnerSelection) MarkExhausted() {
	if s == nil {
		return
	}
	s.Status = WinnerSelectionExhausted
}
