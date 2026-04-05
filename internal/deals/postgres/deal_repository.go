package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type DealRepository struct {
	db *sql.DB
}

func NewDealRepository(db *sql.DB) *DealRepository {
	return &DealRepository{db: db}
}

var _ app.DealRepository = (*DealRepository)(nil)

func (r *DealRepository) Save(ctx context.Context, item *deal.Deal) error {
	const query = `
INSERT INTO deals (
    deal_id,
    customer_id,
    supplier_id,
    auction_id,
    quantity,
    unit_price,
    status,
    type_name,
    created_at,
    confirmed_at,
    contract_number,
    contract_prepared_at,
    contract_signed_at,
    contract_signed_by,
    signature_ref,
    document_url,
    product_id,
    product_name,
    product_description,
    product_category,
    product_weight,
    product_unit,
    product_size,
    product_processing_type,
    product_volume,
    product_origin_country
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)
ON CONFLICT (deal_id) DO UPDATE SET
    customer_id = EXCLUDED.customer_id,
    supplier_id = EXCLUDED.supplier_id,
    auction_id = EXCLUDED.auction_id,
    quantity = EXCLUDED.quantity,
    unit_price = EXCLUDED.unit_price,
    status = EXCLUDED.status,
    type_name = EXCLUDED.type_name,
    created_at = EXCLUDED.created_at,
    confirmed_at = EXCLUDED.confirmed_at,
    contract_number = EXCLUDED.contract_number,
    contract_prepared_at = EXCLUDED.contract_prepared_at,
    contract_signed_at = EXCLUDED.contract_signed_at,
    contract_signed_by = EXCLUDED.contract_signed_by,
    signature_ref = EXCLUDED.signature_ref,
    document_url = EXCLUDED.document_url,
    product_id = EXCLUDED.product_id,
    product_name = EXCLUDED.product_name,
    product_description = EXCLUDED.product_description,
    product_category = EXCLUDED.product_category,
    product_weight = EXCLUDED.product_weight,
    product_unit = EXCLUDED.product_unit,
    product_size = EXCLUDED.product_size,
    product_processing_type = EXCLUDED.product_processing_type,
    product_volume = EXCLUDED.product_volume,
    product_origin_country = EXCLUDED.product_origin_country
`

	dbtx := DBTXFromContext(ctx, r.db)
	snapshot := item.ProductSnapshot()
	contract := item.Contract()

	var (
		contractNumber   sql.NullString
		contractPrepared sql.NullTime
		contractSigned   sql.NullTime
		contractSignedBy sql.NullString
		signatureRef     sql.NullString
		documentURL      sql.NullString
	)
	if contract != nil {
		contractNumber = sql.NullString{String: contract.Number, Valid: contract.Number != ""}
		if contract.PreparedAt != nil {
			contractPrepared = sql.NullTime{Time: *contract.PreparedAt, Valid: true}
		}
		if contract.SignedAt != nil {
			contractSigned = sql.NullTime{Time: *contract.SignedAt, Valid: true}
		}
		contractSignedBy = sql.NullString{String: contract.SignedBy, Valid: contract.SignedBy != ""}
		signatureRef = sql.NullString{String: contract.SignatureRef, Valid: contract.SignatureRef != ""}
		documentURL = sql.NullString{String: contract.DocumentURL, Valid: contract.DocumentURL != ""}
	}

	_, err := dbtx.ExecContext(
		ctx,
		query,
		item.ID(),
		item.CustomerID(),
		item.SupplierID(),
		item.AuctionID(),
		item.Quantity(),
		item.UnitPrice(),
		string(item.Status()),
		string(item.Type()),
		item.CreatedAt(),
		item.ConfirmedAt(),
		contractNumber,
		contractPrepared,
		contractSigned,
		contractSignedBy,
		signatureRef,
		documentURL,
		snapshot.ProductID,
		snapshot.Name,
		snapshot.Description,
		snapshot.Category,
		snapshot.Weight,
		snapshot.Unit,
		snapshot.Size,
		snapshot.ProcessingType,
		snapshot.Volume,
		snapshot.OriginCountry,
	)
	return err
}

func (r *DealRepository) GetByID(ctx context.Context, dealID string) (*deal.Deal, error) {
	const query = `
SELECT deal_id, customer_id, supplier_id, auction_id, quantity, unit_price, status, type_name,
       created_at, confirmed_at, contract_number, contract_prepared_at, contract_signed_at,
       contract_signed_by, signature_ref, document_url,
       product_id, product_name, product_description, product_category, product_weight,
       product_unit, product_size, product_processing_type, product_volume, product_origin_country
FROM deals
WHERE deal_id = $1
`
	return r.getOne(ctx, query, dealID)
}

func (r *DealRepository) GetByAuctionID(ctx context.Context, auctionID string) (*deal.Deal, error) {
	const query = `
SELECT deal_id, customer_id, supplier_id, auction_id, quantity, unit_price, status, type_name,
       created_at, confirmed_at, contract_number, contract_prepared_at, contract_signed_at,
       contract_signed_by, signature_ref, document_url,
       product_id, product_name, product_description, product_category, product_weight,
       product_unit, product_size, product_processing_type, product_volume, product_origin_country
FROM deals
WHERE auction_id = $1
`
	return r.getOne(ctx, query, auctionID)
}

func (r *DealRepository) getOne(ctx context.Context, query string, arg string) (*deal.Deal, error) {
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, arg)

	var (
		id, customerID, supplierID, auctionID, status, typeName                    string
		quantity, unitPrice                                                        int64
		createdAt                                                                  sql.NullTime
		confirmedAt                                                                sql.NullTime
		contractNumber                                                             sql.NullString
		contractPrepared                                                           sql.NullTime
		contractSigned                                                             sql.NullTime
		contractSignedBy                                                           sql.NullString
		signatureRef                                                               sql.NullString
		documentURL                                                                sql.NullString
		productID, productName, productDescription, productCategory, productOrigin string
		productUnit, productSize, productProcessingType                            string
		productWeight, productVolume                                               float64
	)

	if err := row.Scan(
		&id, &customerID, &supplierID, &auctionID, &quantity, &unitPrice, &status, &typeName,
		&createdAt, &confirmedAt, &contractNumber, &contractPrepared, &contractSigned,
		&contractSignedBy, &signatureRef, &documentURL,
		&productID, &productName, &productDescription, &productCategory, &productWeight,
		&productUnit, &productSize, &productProcessingType, &productVolume, &productOrigin,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrDealNotFound
		}
		return nil, err
	}
	if !createdAt.Valid {
		return nil, deal.ErrCreatedAtRequired
	}

	var contract *deal.ContractInfo
	if contractNumber.Valid || contractPrepared.Valid || contractSigned.Valid || contractSignedBy.Valid || signatureRef.Valid || documentURL.Valid {
		contract = &deal.ContractInfo{
			Number:       contractNumber.String,
			SignedBy:     contractSignedBy.String,
			SignatureRef: signatureRef.String,
			DocumentURL:  documentURL.String,
		}
		if contractPrepared.Valid {
			contract.PreparedAt = &contractPrepared.Time
		}
		if contractSigned.Valid {
			contract.SignedAt = &contractSigned.Time
		}
	}

	return deal.Rehydrate(deal.RehydrateParams{
		ID:         id,
		CustomerID: customerID,
		SupplierID: supplierID,
		AuctionID:  auctionID,
		Quantity:   quantity,
		UnitPrice:  unitPrice,
		Status:     deal.DealStatus(status),
		TypeName:   deal.DealType(typeName),
		CreatedAt:  createdAt.Time,
		ConfirmedAt: func() *time.Time {
			if confirmedAt.Valid {
				return &confirmedAt.Time
			}
			return nil
		}(),
		Contract: contract,
		ProductSnapshot: deal.ProductSnapshot{
			ProductID:      productID,
			Name:           productName,
			Description:    productDescription,
			Category:       productCategory,
			Weight:         productWeight,
			Unit:           productUnit,
			Size:           productSize,
			ProcessingType: productProcessingType,
			Volume:         productVolume,
			OriginCountry:  productOrigin,
		},
	})
}
