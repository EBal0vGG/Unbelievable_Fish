package httpapi

import (
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type CreateProjectionRequest struct {
	AuctionID       string             `json:"auction_id"`
	SupplierID      string             `json:"supplier_id"`
	ProductSnapshot ProductSnapshotDTO `json:"product_snapshot"`
	StartPrice      int64              `json:"start_price"`
	PublishedAt     time.Time          `json:"published_at"`
}

type CreateDealFromAuctionWonRequest struct {
	AuctionID       string    `json:"auction_id"`
	WinnerCompanyID string    `json:"winner_company_id"`
	FinalPrice      int64     `json:"final_price"`
	WonAt           time.Time `json:"won_at"`
}

type PrepareContractRequest struct {
	ContractNumber string `json:"contract_number"`
	DocumentURL    string `json:"document_url"`
}

type RequestDealConfirmationRequest struct {
	Stage                 string     `json:"stage"`
	VerificationMethod    string     `json:"verification_method"`
	VerificationTokenHash string     `json:"verification_token_hash,omitempty"`
	SignatureRef          string     `json:"signature_ref,omitempty"`
	Comment               string     `json:"comment,omitempty"`
	ExpiresAt             *time.Time `json:"expires_at,omitempty"`
}

type RejectDealConfirmationRequest struct {
	Reason string `json:"reason,omitempty"`
}

type SignContractRequest struct {
	SignatureRef string `json:"signature_ref"`
}

type RequestPaymentRequest struct {
	InvoiceNumber string     `json:"invoice_number"`
	DueDate       *time.Time `json:"due_date,omitempty"`
}

type MarkDealPaidRequest struct {
	PaymentID   string `json:"payment_id"`
	PaymentType string `json:"payment_type"`
}

type MarkDealShippedRequest struct {
	TrackingNumber string `json:"tracking_number"`
	Carrier        string `json:"carrier"`
}

type CancelDealRequest struct {
	Reason string `json:"reason"`
}

type UpdateDealPriceRequest struct {
	NewPrice int64 `json:"new_price"`
}

type ProductSnapshotDTO struct {
	ProductID      string  `json:"product_id"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	Category       string  `json:"category"`
	Weight         float64 `json:"weight"`
	Unit           string  `json:"unit"`
	Size           string  `json:"size"`
	ProcessingType string  `json:"processing_type"`
	Volume         float64 `json:"volume"`
	OriginCountry  string  `json:"origin_country"`
}

type ContractInfoDTO struct {
	Number       string     `json:"number"`
	PreparedAt   *time.Time `json:"prepared_at,omitempty"`
	SignedAt     *time.Time `json:"signed_at,omitempty"`
	SignedBy     string     `json:"signed_by,omitempty"`
	SignatureRef string     `json:"signature_ref,omitempty"`
	DocumentURL  string     `json:"document_url,omitempty"`
}

type DealConfirmationResponse struct {
	ID                    string     `json:"id"`
	DealID                string     `json:"deal_id"`
	Stage                 string     `json:"stage"`
	RequestedByUserID     string     `json:"requested_by_user_id"`
	RequestedByCompanyID  string     `json:"requested_by_company_id"`
	CounterpartyCompanyID string     `json:"counterparty_company_id"`
	Status                string     `json:"status"`
	VerificationMethod    string     `json:"verification_method"`
	VerificationTokenHash string     `json:"verification_token_hash,omitempty"`
	SignatureRef          string     `json:"signature_ref,omitempty"`
	RequestedAt           time.Time  `json:"requested_at"`
	ApprovedAt            *time.Time `json:"approved_at,omitempty"`
	RejectedAt            *time.Time `json:"rejected_at,omitempty"`
	ExpiresAt             *time.Time `json:"expires_at,omitempty"`
	Comment               string     `json:"comment,omitempty"`
	Reason                string     `json:"reason,omitempty"`
}

type DealResponse struct {
	ID                   string             `json:"id"`
	CustomerID           string             `json:"customer_id"`
	SupplierID           string             `json:"supplier_id"`
	AuctionID            string             `json:"auction_id"`
	Quantity             int64              `json:"quantity"`
	UnitPrice            int64              `json:"unit_price"`
	TotalAmount          int64              `json:"total_amount"`
	Status               string             `json:"status"`
	Type                 string             `json:"type"`
	CreatedAt            time.Time          `json:"created_at"`
	ConfirmedAt          *time.Time         `json:"confirmed_at,omitempty"`
	ContractSignDeadline *time.Time         `json:"contract_sign_deadline,omitempty"`
	PaymentDeadline      *time.Time         `json:"payment_deadline,omitempty"`
	Contract             *ContractInfoDTO   `json:"contract,omitempty"`
	ProductSnapshot      ProductSnapshotDTO `json:"product_snapshot"`
}

type ProjectionResponse struct {
	AuctionID       string             `json:"auction_id"`
	SupplierID      string             `json:"supplier_id"`
	StartPrice      int64              `json:"start_price"`
	PublishedAt     time.Time          `json:"published_at"`
	Status          string             `json:"status"`
	ProductSnapshot ProductSnapshotDTO `json:"product_snapshot"`
}

type ErrorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
}

func (d ProductSnapshotDTO) ToDomain() deal.ProductSnapshot {
	return deal.ProductSnapshot{
		ProductID:      d.ProductID,
		Name:           d.Name,
		Description:    d.Description,
		Category:       d.Category,
		Weight:         d.Weight,
		Unit:           d.Unit,
		Size:           d.Size,
		ProcessingType: d.ProcessingType,
		Volume:         d.Volume,
		OriginCountry:  d.OriginCountry,
	}
}

func NewDealResponse(item *deal.Deal) DealResponse {
	response := DealResponse{
		ID:                   item.ID(),
		CustomerID:           item.CustomerID(),
		SupplierID:           item.SupplierID(),
		AuctionID:            item.AuctionID(),
		Quantity:             item.Quantity(),
		UnitPrice:            item.UnitPrice(),
		TotalAmount:          item.CalculateTotal(),
		Status:               string(item.Status()),
		Type:                 string(item.Type()),
		CreatedAt:            item.CreatedAt(),
		ConfirmedAt:          item.ConfirmedAt(),
		ContractSignDeadline: item.ContractSignDeadline(),
		PaymentDeadline:      item.PaymentDeadline(),
		ProductSnapshot:      NewProductSnapshotDTO(item.ProductSnapshot()),
	}
	if item.Contract() != nil {
		response.Contract = &ContractInfoDTO{
			Number:       item.Contract().Number,
			PreparedAt:   item.Contract().PreparedAt,
			SignedAt:     item.Contract().SignedAt,
			SignedBy:     item.Contract().SignedBy,
			SignatureRef: item.Contract().SignatureRef,
			DocumentURL:  item.Contract().DocumentURL,
		}
	}
	return response
}

func NewProjectionResponse(item *deal.DealProjection) ProjectionResponse {
	return ProjectionResponse{
		AuctionID:       item.AuctionID,
		SupplierID:      item.SupplierID,
		StartPrice:      item.StartPrice,
		PublishedAt:     item.PublishedAt,
		Status:          string(item.Status),
		ProductSnapshot: NewProductSnapshotDTO(item.ProductSnapshot),
	}
}

func NewDealConfirmationResponse(item *deal.DealConfirmation) DealConfirmationResponse {
	return DealConfirmationResponse{
		ID:                    item.ID(),
		DealID:                item.DealID(),
		Stage:                 string(item.Stage()),
		RequestedByUserID:     item.RequestedByUserID(),
		RequestedByCompanyID:  item.RequestedByCompanyID(),
		CounterpartyCompanyID: item.CounterpartyCompanyID(),
		Status:                string(item.Status()),
		VerificationMethod:    string(item.VerificationMethod()),
		VerificationTokenHash: item.VerificationTokenHash(),
		SignatureRef:          item.SignatureRef(),
		RequestedAt:           item.RequestedAt(),
		ApprovedAt:            item.ApprovedAt(),
		RejectedAt:            item.RejectedAt(),
		ExpiresAt:             item.ExpiresAt(),
		Comment:               item.Comment(),
		Reason:                item.Reason(),
	}
}

func NewProductSnapshotDTO(snapshot deal.ProductSnapshot) ProductSnapshotDTO {
	return ProductSnapshotDTO{
		ProductID:      snapshot.ProductID,
		Name:           snapshot.Name,
		Description:    snapshot.Description,
		Category:       snapshot.Category,
		Weight:         snapshot.Weight,
		Unit:           snapshot.Unit,
		Size:           snapshot.Size,
		ProcessingType: snapshot.ProcessingType,
		Volume:         snapshot.Volume,
		OriginCountry:  snapshot.OriginCountry,
	}
}
