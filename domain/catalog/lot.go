package catalog

import (
	"strings"
	"time"
)

type Lot struct {
	lotID           string
	productID       string
	auctionID       string
	sellerCompanyID string

	photo    string
	quantity float64

	startPrice int64
	curPrice   int64
	finalPrice int64

	status LotStatus

	auctionSchedule *AuctionSchedule
}

type AuctionSchedule struct {
	startsAt Instant // это и есть auctionStartDate
}

type Instant struct {
	utc time.Time
}

func NewInstant(t time.Time) Instant {
	return Instant{utc: t.UTC()}
}

func (i Instant) Time() time.Time {
	return i.utc
}

func NewLot(
	lotID, productID, sellerCompanyID string,
	photo string,
	quantity float64,
	startPrice int64,
	auctionSchedule *AuctionSchedule,
) (*Lot, []Event, error) {

	if isBlank(lotID) || isBlank(productID) || isBlank(sellerCompanyID) {
		return nil, nil, ErrInvalidIdentifier
	}
	if quantity <= 0 {
		return nil, nil, ErrInvalidQuantity
	}
	if startPrice <= 0 {
		return nil, nil, ErrInvalidPrice
	}
	if auctionSchedule == nil {
		return nil, nil, ErrInvalidSchedule
	}

	normalizedPhoto := strings.TrimSpace(photo)

	lot := &Lot{
		lotID:           lotID,
		productID:       productID,
		sellerCompanyID: sellerCompanyID,

		auctionID: "",

		photo:    normalizedPhoto,
		quantity: quantity,

		startPrice: startPrice,
		curPrice:   startPrice,
		finalPrice: startPrice,

		status: LotStatusDraft,

		auctionSchedule: auctionSchedule,
	}

	event := LotCreated{
		LotID:           lot.lotID,
		ProductID:       lot.productID,
		SellerCompanyID: lot.sellerCompanyID,
		Photo:           lot.photo,
		Quantity:        lot.quantity,
		Status:          lot.status,
	}

	return lot, []Event{event}, nil
}

func (l *Lot) AssignAuctionID(auctionID string) ([]Event, error) {
	if isBlank(auctionID) {
		return nil, ErrInvalidIdentifier
	}
	if !isBlank(l.auctionID) {
		return nil, ErrAlreadyAssigned
	}
	if l.status != LotStatusDraft {
		return nil, ErrModificationNotAllowed
	}

	l.auctionID = auctionID
	return nil, nil
}

func (l *Lot) ID() string              { return l.lotID }
func (l *Lot) ProductID() string       { return l.productID }
func (l *Lot) SellerCompanyID() string { return l.sellerCompanyID }
func (l *Lot) AuctionID() string       { return l.auctionID }
func (l *Lot) Photo() string           { return l.photo }
func (l *Lot) Quantity() float64       { return l.quantity }
func (l *Lot) Status() LotStatus       { return l.status }
func (l *Lot) StartPrice() int64       { return l.startPrice }
func (l *Lot) CurPrice() int64         { return l.curPrice }
func (l *Lot) FinalPrice() int64       { return l.finalPrice }

func (l *Lot) Publish(productIsPublished bool, product ProductSnapshot) ([]Event, error) {
	if !l.canTransition(LotStatusPublished) {
		return nil, ErrForbiddenStateTransition
	}

	if !productIsPublished {
		return nil, ErrPublishingRuleViolation
	}

	if isBlank(l.auctionID) {
		return nil, ErrAuctionIDRequired
	}

	l.status = LotStatusPublished

	event := LotPublished{
		LotID:           l.lotID,
		AuctionID:       l.auctionID,
		SellerCompanyID: l.sellerCompanyID,
		ProductID:       l.productID,
		Product:         product,
		StartPrice:      l.startPrice,
		Status:          l.status,
	}

	return []Event{event}, nil
}

func (l *Lot) Unpublish() ([]Event, error) {
	// Unpublish transitions a published lot to CANCELLED.
	if !l.canTransition(LotStatusCancelled) {
		return nil, ErrForbiddenStateTransition
	}

	l.status = LotStatusCancelled

	event := LotUnpublished{
		LotID:  l.lotID,
		Status: l.status,
	}

	return []Event{event}, nil
}

func (l *Lot) Close(finalPrice int64) ([]Event, error) {
	if !l.canTransition(LotStatusClosed) {
		return nil, ErrForbiddenStateTransition
	}
	if finalPrice <= 0 {
		return nil, ErrInvalidPrice
	}

	l.status = LotStatusClosed
	l.finalPrice = finalPrice

	event := LotClosed{
		LotID:      l.lotID,
		FinalPrice: finalPrice,
		Status:     l.status,
	}

	return []Event{event}, nil
}

func (l *Lot) UpdateCurrentPrice(amount int64) ([]Event, error) {
	if l.status != LotStatusPublished {
		return nil, ErrModificationNotAllowed
	}
	if amount <= 0 {
		return nil, ErrInvalidPrice
	}

	l.curPrice = amount

	event := LotPriceUpdated{
		LotID:        l.lotID,
		AuctionID:    l.auctionID,
		CurrentPrice: l.curPrice,
		Status:       l.status,
	}

	return []Event{event}, nil
}

func (l *Lot) canTransition(to LotStatus) bool {
	next, ok := lotTransitions[l.status]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}
