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
	LotStatusSold      LotStatus = "SOLD"
	LotStatusCancelled LotStatus = "CANCELLED"
)

func (s LotStatus) IsValid() bool {
	switch s {
	case LotStatusDraft, LotStatusPublished, LotStatusSold, LotStatusCancelled:
		return true
	default:
		return false
	}
}
