package deal

import "time"

type DealConfirmationStage string

const (
	DealConfirmationStageConfirmed DealConfirmationStage = "confirmed"
	DealConfirmationStagePaid      DealConfirmationStage = "paid"
	DealConfirmationStageShipped   DealConfirmationStage = "shipped"
	DealConfirmationStageCompleted DealConfirmationStage = "completed"
	DealConfirmationStageCancelled DealConfirmationStage = "cancelled"
)

type DealConfirmationStatus string

const (
	DealConfirmationStatusPending  DealConfirmationStatus = "pending"
	DealConfirmationStatusApproved DealConfirmationStatus = "approved"
	DealConfirmationStatusRejected DealConfirmationStatus = "rejected"
	DealConfirmationStatusExpired  DealConfirmationStatus = "expired"
)

type VerificationMethod string

const (
	VerificationMethodManual    VerificationMethod = "manual"
	VerificationMethodEmail     VerificationMethod = "email"
	VerificationMethodSignature VerificationMethod = "signature"
)

type DealConfirmation struct {
	id                    string
	dealID                string
	stage                 DealConfirmationStage
	requestedByUserID     string
	requestedByCompanyID  string
	counterpartyCompanyID string
	status                DealConfirmationStatus
	verificationMethod    VerificationMethod
	verificationTokenHash string
	signatureRef          string
	requestedAt           time.Time
	approvedAt            *time.Time
	rejectedAt            *time.Time
	expiresAt             *time.Time
	comment               string
	reason                string
}

type DealConfirmationParams struct {
	ID                    string
	DealID                string
	Stage                 DealConfirmationStage
	RequestedByUserID     string
	RequestedByCompanyID  string
	CounterpartyCompanyID string
	Status                DealConfirmationStatus
	VerificationMethod    VerificationMethod
	VerificationTokenHash string
	SignatureRef          string
	RequestedAt           time.Time
	ApprovedAt            *time.Time
	RejectedAt            *time.Time
	ExpiresAt             *time.Time
	Comment               string
	Reason                string
}

func NewDealConfirmation(params DealConfirmationParams) (*DealConfirmation, error) {
	if params.ID == "" {
		params.ID = generateConfirmationID()
	}
	if params.Status == "" {
		params.Status = DealConfirmationStatusPending
	}
	if params.RequestedAt.IsZero() {
		params.RequestedAt = time.Now().UTC()
	}

	item := &DealConfirmation{
		id:                    params.ID,
		dealID:                params.DealID,
		stage:                 params.Stage,
		requestedByUserID:     params.RequestedByUserID,
		requestedByCompanyID:  params.RequestedByCompanyID,
		counterpartyCompanyID: params.CounterpartyCompanyID,
		status:                params.Status,
		verificationMethod:    params.VerificationMethod,
		verificationTokenHash: params.VerificationTokenHash,
		signatureRef:          params.SignatureRef,
		requestedAt:           params.RequestedAt,
		approvedAt:            params.ApprovedAt,
		rejectedAt:            params.RejectedAt,
		expiresAt:             params.ExpiresAt,
		comment:               params.Comment,
		reason:                params.Reason,
	}
	if err := item.Validate(); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *DealConfirmation) ID() string                     { return c.id }
func (c *DealConfirmation) DealID() string                 { return c.dealID }
func (c *DealConfirmation) Stage() DealConfirmationStage   { return c.stage }
func (c *DealConfirmation) RequestedByUserID() string      { return c.requestedByUserID }
func (c *DealConfirmation) RequestedByCompanyID() string   { return c.requestedByCompanyID }
func (c *DealConfirmation) CounterpartyCompanyID() string  { return c.counterpartyCompanyID }
func (c *DealConfirmation) Status() DealConfirmationStatus { return c.status }
func (c *DealConfirmation) VerificationMethod() VerificationMethod {
	return c.verificationMethod
}
func (c *DealConfirmation) VerificationTokenHash() string { return c.verificationTokenHash }
func (c *DealConfirmation) SignatureRef() string          { return c.signatureRef }
func (c *DealConfirmation) RequestedAt() time.Time        { return c.requestedAt }
func (c *DealConfirmation) ApprovedAt() *time.Time        { return c.approvedAt }
func (c *DealConfirmation) RejectedAt() *time.Time        { return c.rejectedAt }
func (c *DealConfirmation) ExpiresAt() *time.Time         { return c.expiresAt }
func (c *DealConfirmation) Comment() string               { return c.comment }
func (c *DealConfirmation) Reason() string                { return c.reason }

func (c *DealConfirmation) Validate() error {
	if c.id == "" {
		return ErrConfirmationIDRequired
	}
	if c.dealID == "" {
		return ErrConfirmationDealIDRequired
	}
	if !c.stage.Valid() {
		return ErrConfirmationStageRequired
	}
	if c.requestedByUserID == "" {
		return ErrRequestedByUserRequired
	}
	if c.requestedByCompanyID == "" {
		return ErrRequestedByCompanyRequired
	}
	if c.counterpartyCompanyID == "" {
		return ErrCounterpartyCompanyRequired
	}
	if c.requestedByCompanyID == c.counterpartyCompanyID {
		return ErrCounterpartyRequired
	}
	if !c.status.Valid() {
		return ErrConfirmationStatusRequired
	}
	if !c.verificationMethod.Valid() {
		return ErrVerificationMethodRequired
	}
	if c.requestedAt.IsZero() {
		return ErrRequestedAtRequired
	}
	return nil
}

func (c *DealConfirmation) Approve(approvedByCompanyID, approvedByUserID string, approvedAt time.Time) ([]Event, error) {
	if c.status != DealConfirmationStatusPending {
		return nil, ErrConfirmationNotPending
	}
	if approvedByCompanyID == "" || approvedByUserID == "" {
		return nil, ErrCounterpartyRequired
	}
	if approvedByCompanyID == c.requestedByCompanyID {
		return nil, ErrCounterpartyRequired
	}
	if approvedByCompanyID != c.counterpartyCompanyID {
		return nil, ErrNotDealParticipant
	}
	if c.expiresAt != nil && approvedAt.After(*c.expiresAt) {
		c.status = DealConfirmationStatusExpired
		return nil, ErrConfirmationExpired
	}

	c.status = DealConfirmationStatusApproved
	c.approvedAt = &approvedAt

	return []Event{
		DealConfirmationApproved{
			ConfirmationID:      c.id,
			DealID:              c.dealID,
			Stage:               c.stage,
			ApprovedByUserID:    approvedByUserID,
			ApprovedByCompanyID: approvedByCompanyID,
			ApprovedAt:          approvedAt,
		},
	}, nil
}

func (c *DealConfirmation) Reject(rejectedByCompanyID, rejectedByUserID, reason string, rejectedAt time.Time) ([]Event, error) {
	if c.status != DealConfirmationStatusPending {
		return nil, ErrConfirmationNotPending
	}
	if rejectedByCompanyID == "" || rejectedByUserID == "" {
		return nil, ErrCounterpartyRequired
	}
	if rejectedByCompanyID == c.requestedByCompanyID {
		return nil, ErrCounterpartyRequired
	}
	if rejectedByCompanyID != c.counterpartyCompanyID {
		return nil, ErrNotDealParticipant
	}
	if c.expiresAt != nil && rejectedAt.After(*c.expiresAt) {
		c.status = DealConfirmationStatusExpired
		return nil, ErrConfirmationExpired
	}

	c.status = DealConfirmationStatusRejected
	c.rejectedAt = &rejectedAt
	c.reason = reason

	return []Event{
		DealConfirmationRejected{
			ConfirmationID:      c.id,
			DealID:              c.dealID,
			Stage:               c.stage,
			RejectedByUserID:    rejectedByUserID,
			RejectedByCompanyID: rejectedByCompanyID,
			RejectedAt:          rejectedAt,
			Reason:              reason,
		},
	}, nil
}

func (s DealConfirmationStage) Valid() bool {
	switch s {
	case DealConfirmationStageConfirmed,
		DealConfirmationStagePaid,
		DealConfirmationStageShipped,
		DealConfirmationStageCompleted,
		DealConfirmationStageCancelled:
		return true
	default:
		return false
	}
}

func (s DealConfirmationStatus) Valid() bool {
	switch s {
	case DealConfirmationStatusPending,
		DealConfirmationStatusApproved,
		DealConfirmationStatusRejected,
		DealConfirmationStatusExpired:
		return true
	default:
		return false
	}
}

func (m VerificationMethod) Valid() bool {
	switch m {
	case VerificationMethodManual, VerificationMethodEmail, VerificationMethodSignature:
		return true
	default:
		return false
	}
}

func generateConfirmationID() string {
	return "dcf_" + time.Now().UTC().Format("20060102150405.000000000")
}
