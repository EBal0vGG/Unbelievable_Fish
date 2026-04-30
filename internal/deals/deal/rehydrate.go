package deal

import "time"

type RehydrateParams struct {
	ID                   string
	CustomerID           string
	SupplierID           string
	AuctionID            string
	Quantity             int64
	UnitPrice            int64
	Status               DealStatus
	TypeName             DealType
	CreatedAt            time.Time
	ConfirmedAt          *time.Time
	ContractSignDeadline *time.Time
	PaymentDeadline      *time.Time
	Contract             *ContractInfo
	ProductSnapshot      ProductSnapshot
}

func Rehydrate(params RehydrateParams) (*Deal, error) {
	item := &Deal{
		id:                   params.ID,
		customerID:           params.CustomerID,
		supplierID:           params.SupplierID,
		auctionID:            params.AuctionID,
		quantity:             params.Quantity,
		unitPrice:            params.UnitPrice,
		status:               params.Status,
		typeName:             params.TypeName,
		createdAt:            params.CreatedAt,
		confirmedAt:          params.ConfirmedAt,
		contractSignDeadline: params.ContractSignDeadline,
		paymentDeadline:      params.PaymentDeadline,
		contract:             params.Contract,
		productSnapshot:      params.ProductSnapshot,
	}
	if err := item.Validate(); err != nil {
		return nil, err
	}
	return item, nil
}
