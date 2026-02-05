package catalog

type Product struct {
	productID       string
	sellerCompanyID string
	title           string
	species         string
	processingType  ProcessingType
	packagingType   PackagingType
	quantity        int64
	size            string
	status          ProductStatus
}

func NewProduct(productID, sellerCompanyID, title, species string, processingType ProcessingType, packagingType PackagingType, size string) (*Product, []Event, error) {
	if isBlank(productID) || isBlank(sellerCompanyID) {
		return nil, nil, ErrInvalidIdentifier
	}
	if processingType != "" && !processingType.IsValid() {
		return nil, nil, ErrInvalidEnum
	}
	if packagingType != "" && !packagingType.IsValid() {
		return nil, nil, ErrInvalidEnum
	}

	product := &Product{
		productID:       productID,
		sellerCompanyID: sellerCompanyID,
		title:           title,
		species:         species,
		processingType:  processingType,
		packagingType:   packagingType,
		size:            size,
		status:          ProductStatusDraft,
	}

	event := ProductCreated{
		ProductID:       product.productID,
		SellerCompanyID: product.sellerCompanyID,
		Title:           product.title,
		Species:         product.species,
		ProcessingType:  product.processingType,
		PackagingType:   product.packagingType,
		Size:            product.size,
		Status:          product.status,
	}

	return product, []Event{event}, nil
}

func (p *Product) ID() string {
	return p.productID
}

func (p *Product) SellerCompanyID() string {
	return p.sellerCompanyID
}

func (p *Product) Title() string {
	return p.title
}

func (p *Product) Species() string {
	return p.species
}

func (p *Product) ProcessingType() ProcessingType {
	return p.processingType
}

func (p *Product) PackagingType() PackagingType {
	return p.packagingType
}

func (p *Product) Size() string {
	return p.size
}

func (p *Product) Status() ProductStatus {
	return p.status
}

func (p *Product) UpdateDetails(title, species string, processingType ProcessingType, packagingType PackagingType, size string) ([]Event, error) {
	if p.status == ProductStatusPublished {
		return nil, ErrModificationNotAllowed
	}
	if processingType != "" && !processingType.IsValid() {
		return nil, ErrInvalidEnum
	}
	if packagingType != "" && !packagingType.IsValid() {
		return nil, ErrInvalidEnum
	}

	p.title = title
	p.species = species
	p.processingType = processingType
	p.packagingType = packagingType
	p.size = size

	event := ProductUpdated{
		ProductID:       p.productID,
		SellerCompanyID: p.sellerCompanyID,
		Title:           p.title,
		Species:         p.species,
		ProcessingType:  p.processingType,
		PackagingType:   p.packagingType,
		Size:            p.size,
		Status:          p.status,
	}

	return []Event{event}, nil
}

func (p *Product) Publish() ([]Event, error) {
	if !p.canTransition(ProductStatusPublished) {
		return nil, ErrForbiddenStateTransition
	}
	if !p.isPublishable() {
		return nil, ErrPublishingRuleViolation
	}

	p.status = ProductStatusPublished
	event := ProductPublished{
		ProductID: p.productID,
		Status:    p.status,
	}

	return []Event{event}, nil
}

func (p *Product) Unpublish() ([]Event, error) {
	if !p.canTransition(ProductStatusUnpublished) {
		return nil, ErrForbiddenStateTransition
	}

	p.status = ProductStatusUnpublished
	event := ProductUnpublished{
		ProductID: p.productID,
		Status:    p.status,
	}

	return []Event{event}, nil
}

func (p *Product) isPublishable() bool {
	if isBlank(p.productID) || isBlank(p.sellerCompanyID) {
		return false
	}
	if isBlank(p.title) || isBlank(p.species) || isBlank(p.size) {
		return false
	}
	if !p.processingType.IsValid() || !p.packagingType.IsValid() {
		return false
	}
	return true
}

func (p *Product) canTransition(to ProductStatus) bool {
	next, ok := productTransitions[p.status]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}
