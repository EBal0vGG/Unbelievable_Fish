package catalog

type Product struct {
	productID string
	fishID    string

	weight int64
	size   string
	unit   string

	processingType ProcessingType
	status         ProductStatus
}

func NewProduct(
	productID, fishID string,
	weight int64,
	unit string,
	size string,
	processingType ProcessingType,
) (*Product, []Event, error) {

	if isBlank(productID) || isBlank(fishID) || isBlank(unit) {
		return nil, nil, ErrInvalidIdentifier
	}

	if weight <= 0 {
		return nil, nil, ErrInvalidWeight
	}

	p := &Product{
		productID:      productID,
		fishID:         fishID,
		weight:         weight,
		unit:           unit,
		size:           size,
		processingType: processingType,
		status:         ProductStatusDraft,
	}

	ev := ProductCreated{
		ProductID:      p.productID,
		FishID:         p.fishID,
		Weight:         p.weight,
		Unit:           p.unit,
		Size:           p.size,
		ProcessingType: p.processingType,
		Status:         p.status,
	}

	return p, []Event{ev}, nil
}

func (p *Product) ID() string                     { return p.productID }
func (p *Product) FishID() string                 { return p.fishID }
func (p *Product) Weight() int64                  { return p.weight }
func (p *Product) Unit() string                   { return p.unit }
func (p *Product) Size() string                   { return p.size }
func (p *Product) ProcessingType() ProcessingType { return p.processingType }
func (p *Product) Status() ProductStatus          { return p.status }

func (p *Product) Update(
	fishID string,
	weight int64,
	unit string,
	size string,
	processingType ProcessingType,
) ([]Event, error) {

	if p.status != ProductStatusDraft {
		return nil, ErrModificationNotAllowed
	}
	if isBlank(fishID) || isBlank(unit) {
		return nil, ErrInvalidIdentifier
	}

	if weight <= 0 {
		return nil, ErrInvalidWeight
	}

	p.fishID = fishID
	p.weight = weight
	p.unit = unit
	p.size = size
	p.processingType = processingType

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
