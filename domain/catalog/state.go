package catalog

type ProductStatus string

const (
	ProductStatusDraft       ProductStatus = "DRAFT"
	ProductStatusPublished   ProductStatus = "PUBLISHED"
	ProductStatusUnpublished ProductStatus = "UNPUBLISHED"
)

func (s ProductStatus) IsValid() bool {
	switch s {
	case ProductStatusDraft, ProductStatusPublished, ProductStatusUnpublished:
		return true
	default:
		return false
	}
}

type LotStatus string

const (
	LotStatusDraft     LotStatus = "DRAFT"
	LotStatusPublished LotStatus = "PUBLISHED"
	LotStatusClosed    LotStatus = "CLOSED"
	LotStatusCancelled LotStatus = "CANCELLED"
)

func (s LotStatus) IsValid() bool {
	switch s {
	case LotStatusDraft, LotStatusPublished, LotStatusClosed, LotStatusCancelled:
		return true
	default:
		return false
	}
}
