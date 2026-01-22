package catalog

type Lot struct {
	lotID           string
	productID       string
	sellerCompanyID string
	quantity        int64
	unit            Unit
	deliveryTerms   string
	storageTerms    string
	status          LotStatus
	dealID          string
}

func NewLot(lotID, productID, sellerCompanyID string, quantity int64, unit Unit, deliveryTerms, storageTerms string) (*Lot, []Event, error) {
	if isBlank(lotID) || isBlank(productID) || isBlank(sellerCompanyID) {
		return nil, nil, ErrInvalidIdentifier
	}
	if quantity <= 0 {
		return nil, nil, ErrInvalidQuantity
	}
	if !unit.IsValid() {
		return nil, nil, ErrInvalidEnum
	}
	if isBlank(deliveryTerms) || isBlank(storageTerms) {
		return nil, nil, ErrPublishingRuleViolation
	}

	lot := &Lot{
		lotID:           lotID,
		productID:       productID,
		sellerCompanyID: sellerCompanyID,
		quantity:        quantity,
		unit:            unit,
		deliveryTerms:   deliveryTerms,
		storageTerms:    storageTerms,
		status:          LotStatusDraft,
		dealID:          "",
	}

	event := LotCreated{
		LotID:           lot.lotID,
		ProductID:       lot.productID,
		SellerCompanyID: lot.sellerCompanyID,
		Quantity:        lot.quantity,
		Unit:            lot.unit,
		DeliveryTerms:   lot.deliveryTerms,
		StorageTerms:    lot.storageTerms,
		Status:          lot.status,
	}

	return lot, []Event{event}, nil
}

func (l *Lot) ID() string {
	return l.lotID
}

func (l *Lot) ProductID() string {
	return l.productID
}

func (l *Lot) SellerCompanyID() string {
	return l.sellerCompanyID
}

func (l *Lot) Quantity() int64 {
	return l.quantity
}

func (l *Lot) Unit() Unit {
	return l.unit
}

func (l *Lot) DeliveryTerms() string {
	return l.deliveryTerms
}

func (l *Lot) StorageTerms() string {
	return l.storageTerms
}

func (l *Lot) Status() LotStatus {
	return l.status
}

func (l *Lot) DealID() string {
	return l.dealID
}

func (l *Lot) Publish(productIsPublished bool) ([]Event, error) {
	if !l.canTransition(LotStatusPublished) {
		return nil, ErrForbiddenStateTransition
	}
	if !productIsPublished {
		return nil, ErrPublishingRuleViolation
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

func (l *Lot) MarkSold(dealID string) ([]Event, error) {
	if !l.canTransition(LotStatusSold) {
		return nil, ErrForbiddenStateTransition
	}
	if isBlank(dealID) {
		return nil, ErrInvalidIdentifier
	}

	l.status = LotStatusSold
	l.dealID = dealID
	event := LotSold{
		LotID:  l.lotID,
		DealID: l.dealID,
		Status: l.status,
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
