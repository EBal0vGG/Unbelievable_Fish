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
		LotStatusClosed:    {},
		LotStatusCancelled: {},
	},
}

func canTransitionLot(from, to LotStatus) bool {
	next, ok := lotTransitions[from]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}
