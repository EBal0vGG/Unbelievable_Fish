package catalog

import "time"

type Lot struct {
	lotID           string
	productID       string
	auctionID       string
	sellerCompanyID string

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
	startPrice int64,
	auctionSchedule *AuctionSchedule,
) (*Lot, []Event, error) {

	if isBlank(lotID) || isBlank(productID) || isBlank(sellerCompanyID) {
		return nil, nil, ErrInvalidIdentifier
	}
	if startPrice <= 0 {
		return nil, nil, ErrInvalidPrice
	}
	if auctionSchedule == nil {
		return nil, nil, ErrInvalidSchedule
	}

	lot := &Lot{
		lotID:           lotID,
		productID:       productID,
		sellerCompanyID: sellerCompanyID,

		auctionID: "",

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
func (l *Lot) Status() LotStatus       { return l.status }
func (l *Lot) StartPrice() int64       { return l.startPrice }
func (l *Lot) CurPrice() int64         { return l.curPrice }
func (l *Lot) FinalPrice() int64       { return l.finalPrice }

func (l *Lot) Publish(productIsPublished bool) ([]Event, error) {
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
		LotID:     l.lotID,
		ProductID: l.productID,
		Status:    l.status,
	}

	return []Event{event}, nil
}

func (l *Lot) Unpublish() ([]Event, error) {
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

/*func (l *Lot) MarkSold(dealID string, finalPrice int64) ([]Event, error) {
	if !l.canTransition(LotStatusSold) {
		return nil, ErrForbiddenStateTransition
	}
	if isBlank(dealID) {
		return nil, ErrInvalidIdentifier
	}
	if finalPrice <= 0 {
		return nil, ErrInvalidPrice
	}

	l.status = LotStatusSold
	l.dealID = dealID
	l.finalPrice = finalPrice

	event := LotSold{
		LotID:  l.lotID,
		DealID: l.dealID,
		Status: l.status,
	}

	return []Event{event}, nil
}*/

func (l *Lot) canTransition(to LotStatus) bool {
	next, ok := lotTransitions[l.status]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}
