package catalog

import (
	"strings"
	"time"
)

type Lot struct {
	lotID string

	productID       string
	sellerCompanyID string
	auctionID       string

	photo string

	quantity float64

	status LotStatus

	product    ProductSnapshot
	startPrice int64
	finalPrice int64
	curPrice   int64

	auctionSchedule *AuctionSchedule
}

func NewLot(
	lotID, productID, sellerCompanyID, photo string,
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
	if auctionSchedule == nil || !auctionSchedule.IsValid() {
		return nil, nil, ErrInvalidSchedule
	}

	lot := &Lot{
		lotID:           lotID,
		productID:       productID,
		sellerCompanyID: sellerCompanyID,
		photo:           strings.TrimSpace(photo),
		quantity:        quantity,
		startPrice:      startPrice,
		curPrice:        startPrice,
		finalPrice:      startPrice,
		status:          LotStatusDraft,
		auctionSchedule: auctionSchedule,
	}

	ev := LotCreated{
		LotID:           lot.lotID,
		ProductID:       lot.productID,
		SellerCompanyID: lot.sellerCompanyID,
		Photo:           lot.photo,
		Quantity:        lot.quantity,
		Status:          lot.status,
	}

	return lot, []Event{ev}, nil
}

func (l *Lot) ID() string              { return l.lotID }
func (l *Lot) ProductID() string       { return l.productID }
func (l *Lot) SellerID() string        { return l.sellerCompanyID }
func (l *Lot) SellerCompanyID() string { return l.sellerCompanyID }
func (l *Lot) AuctionID() string       { return l.auctionID }
func (l *Lot) Photo() string           { return l.photo }
func (l *Lot) Quantity() float64       { return l.quantity }
func (l *Lot) Status() LotStatus       { return l.status }
func (l *Lot) StartPrice() int64       { return l.startPrice }
func (l *Lot) FinalPrice() int64       { return l.finalPrice }
func (l *Lot) CurPrice() int64         { return l.curPrice }
func (l *Lot) Product() ProductSnapshot { return l.product }
func (l *Lot) AuctionSchedule() *AuctionSchedule { return l.auctionSchedule }
func (l *Lot) AuctionStartsAt() time.Time        { return l.auctionSchedule.StartsAt() }
func (l *Lot) AuctionEndsAt() time.Time          { return l.auctionSchedule.EndsAt() }

func (l *Lot) AssignAuctionID(auctionID string) ([]Event, error) {
	if isBlank(auctionID) {
		return nil, ErrInvalidIdentifier
	}
	if !isBlank(l.auctionID) {
		return nil, ErrAlreadyAssigned
	}

	l.auctionID = auctionID

	ev := LotAuctionLinked{
		LotID:     l.lotID,
		AuctionID: l.auctionID,
	}

	return []Event{ev}, nil
}

func (l *Lot) Publish(productIsPublished bool, snapshot ProductSnapshot) ([]Event, error) {
	if !productIsPublished {
		return nil, ErrPublishingRuleViolation
	}
	if !canTransitionLot(l.status, LotStatusPublished) {
		return nil, ErrForbiddenStateTransition
	}

	l.status = LotStatusPublished
	l.product = snapshot

	ev := LotPublished{
		LotID:           l.lotID,
		AuctionID:       l.auctionID,
		SellerCompanyID: l.sellerCompanyID,
		ProductID:       l.productID,
		Product:         l.product,
		StartPrice:      l.startPrice,
		AuctionStartsAt: l.auctionSchedule.StartsAt(),
		AuctionEndsAt:   l.auctionSchedule.EndsAt(),
		Status:          l.status,
	}

	return []Event{ev}, nil
}

func (l *Lot) Unpublish() ([]Event, error) {
	if !canTransitionLot(l.status, LotStatusCancelled) {
		return nil, ErrForbiddenStateTransition
	}

	l.status = LotStatusCancelled

	ev := LotUnpublished{
		LotID:  l.lotID,
		Status: l.status,
	}

	return []Event{ev}, nil
}

func (l *Lot) Close(finalPrice int64) ([]Event, error) {
	if !canTransitionLot(l.status, LotStatusClosed) {
		return nil, ErrForbiddenStateTransition
	}
	if finalPrice <= 0 {
		return nil, ErrInvalidPrice
	}

	l.status = LotStatusClosed
	l.finalPrice = finalPrice

	ev := LotClosed{
		LotID:      l.lotID,
		FinalPrice: l.finalPrice,
		Status:     l.status,
	}

	return []Event{ev}, nil
}

func (l *Lot) UpdateCurrentPrice(currentPrice int64) ([]Event, error) {
	if l.status != LotStatusPublished {
		return nil, ErrForbiddenStateTransition
	}
	if currentPrice <= 0 {
		return nil, ErrInvalidPrice
	}

	l.curPrice = currentPrice

	ev := LotPriceUpdated{
		LotID:        l.lotID,
		AuctionID:    l.auctionID,
		CurrentPrice: l.curPrice,
		Status:       l.status,
	}

	return []Event{ev}, nil
}
