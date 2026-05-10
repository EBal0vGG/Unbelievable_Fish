package catalog

import "strings"

type Product struct {
	productID       string
	fishID          string
	sellerCompanyID string

	weight float64
	size   string
	unit   string

	processingType ProcessingType
	status         ProductStatus
}

// NewProduct creates a draft product; sellerCompanyID is the owning company (required).
func NewProduct(
	productID, fishID, sellerCompanyID string,
	weight float64,
	unit string,
	size string,
	processingType ProcessingType,
) (*Product, []Event, error) {
	if isBlank(sellerCompanyID) {
		return nil, nil, ErrInvalidIdentifier
	}
	p, err := restoreProduct(productID, fishID, sellerCompanyID, weight, unit, size, processingType, ProductStatusDraft)
	if err != nil {
		return nil, nil, err
	}

	ev := ProductCreated{
		ProductID:       p.productID,
		SellerCompanyID: p.sellerCompanyID,
		FishID:          p.fishID,
		Weight:          p.weight,
		Unit:            p.unit,
		Size:            p.size,
		ProcessingType:  p.processingType,
		Status:          p.status,
	}

	return p, []Event{ev}, nil
}

// RestoreProduct rebuilds a product from persistence (sellerCompanyID may be empty for legacy rows).
func RestoreProduct(
	productID, fishID, sellerCompanyID string,
	weight float64,
	unit string,
	size string,
	processingType ProcessingType,
	status ProductStatus,
) (*Product, error) {
	return restoreProduct(productID, fishID, sellerCompanyID, weight, unit, size, processingType, status)
}

func restoreProduct(
	productID, fishID, sellerCompanyID string,
	weight float64,
	unit string,
	size string,
	processingType ProcessingType,
	status ProductStatus,
) (*Product, error) {
	if isBlank(productID) || isBlank(fishID) {
		return nil, ErrInvalidIdentifier
	}

	if isBlank(unit) {
		return nil, ErrInvalidUnit
	}
	if isBlank(string(processingType)) {
		return nil, ErrInvalidEnum
	}
	if weight <= 0 {
		return nil, ErrInvalidWeight
	}

	return &Product{
		productID:       productID,
		fishID:          fishID,
		sellerCompanyID: strings.TrimSpace(sellerCompanyID),
		weight:          weight,
		unit:            strings.TrimSpace(unit),
		size:            size,
		processingType:  ProcessingType(strings.TrimSpace(string(processingType))),
		status:          status,
	}, nil
}

func (p *Product) ID() string { return p.productID }
func (p *Product) FishID() string {
	return p.fishID
}
func (p *Product) SellerCompanyID() string {
	return p.sellerCompanyID
}
func (p *Product) Weight() float64 { return p.weight }
func (p *Product) Unit() string    { return p.unit }
func (p *Product) Size() string    { return p.size }
func (p *Product) ProcessingType() ProcessingType {
	return p.processingType
}
func (p *Product) Status() ProductStatus { return p.status }

func (p *Product) Update(
	fishID string,
	weight float64,
	unit string,
	size string,
	processingType ProcessingType,
) ([]Event, error) {

	if p.status != ProductStatusDraft {
		return nil, ErrModificationNotAllowed
	}
	if isBlank(fishID) {
		return nil, ErrInvalidIdentifier
	}

	if isBlank(unit) {
		return nil, ErrInvalidUnit
	}
	if isBlank(string(processingType)) {
		return nil, ErrInvalidEnum
	}
	if weight <= 0 {
		return nil, ErrInvalidWeight
	}

	p.fishID = fishID
	p.weight = weight
	p.unit = strings.TrimSpace(unit)
	p.size = size
	p.processingType = ProcessingType(strings.TrimSpace(string(processingType)))

	ev := ProductUpdated{
		ProductID:      p.productID,
		FishID:         p.fishID,
		Weight:         p.weight,
		Unit:           p.unit,
		Size:           p.size,
		ProcessingType: p.processingType,
		Status:         p.status,
	}

	return []Event{ev}, nil
}

func (p *Product) Publish() ([]Event, error) {
	if p.status != ProductStatusDraft {
		return nil, ErrForbiddenStateTransition
	}

	p.status = ProductStatusPublished

	ev := ProductPublished{
		ProductID: p.productID,
		Status:    p.status,
	}

	return []Event{ev}, nil
}

func (p *Product) Unpublish() ([]Event, error) {
	if p.status != ProductStatusPublished {
		return nil, ErrForbiddenStateTransition
	}

	p.status = ProductStatusDraft

	ev := ProductUnpublished{
		ProductID: p.productID,
		Status:    p.status,
	}

	return []Event{ev}, nil
}
