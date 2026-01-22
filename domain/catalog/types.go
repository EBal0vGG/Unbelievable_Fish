package catalog

type ProcessingType string

const (
	ProcessingFrozen  ProcessingType = "frozen"
	ProcessingChilled ProcessingType = "chilled"
	ProcessingLive    ProcessingType = "live"
)

func (t ProcessingType) IsValid() bool {
	switch t {
	case ProcessingFrozen, ProcessingChilled, ProcessingLive:
		return true
	default:
		return false
	}
}

type PackagingType string

const (
	PackagingBox    PackagingType = "box"
	PackagingBag    PackagingType = "bag"
	PackagingPallet PackagingType = "pallet"
)

func (t PackagingType) IsValid() bool {
	switch t {
	case PackagingBox, PackagingBag, PackagingPallet:
		return true
	default:
		return false
	}
}

type Unit string

const (
	UnitKg  Unit = "kg"
	UnitTon Unit = "ton"
	UnitPcs Unit = "pcs"
)

func (u Unit) IsValid() bool {
	switch u {
	case UnitKg, UnitTon, UnitPcs:
		return true
	default:
		return false
	}
}
