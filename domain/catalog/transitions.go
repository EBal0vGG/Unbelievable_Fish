package catalog

var productTransitions = map[ProductStatus]map[ProductStatus]struct{}{
	ProductStatusDraft: {
		ProductStatusPublished: {},
	},
	ProductStatusPublished: {
		ProductStatusUnpublished: {},
	},
	ProductStatusUnpublished: {
		ProductStatusPublished: {},
	},
}

var lotTransitions = map[LotStatus]map[LotStatus]struct{}{
	LotStatusDraft: {
		LotStatusPublished: {},
	},
	LotStatusPublished: {
		LotStatusSold:      {},
		LotStatusCancelled: {},
	},
}
