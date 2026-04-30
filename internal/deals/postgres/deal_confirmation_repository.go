package postgres

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type DealConfirmationRepository struct {
	db *sql.DB
}

func NewDealConfirmationRepository(db *sql.DB) *DealConfirmationRepository {
	return &DealConfirmationRepository{db: db}
}

var _ app.DealConfirmationRepository = (*DealConfirmationRepository)(nil)

func (r *DealConfirmationRepository) Save(ctx context.Context, item *deal.DealConfirmation) error {
	const query = `
INSERT INTO deal_confirmations (
    confirmation_id,
    deal_id,
    stage,
    requested_by_user_id,
    requested_by_company_id,
    counterparty_company_id,
    status,
    verification_method,
    verification_token_hash,
    signature_ref,
    requested_at,
    approved_at,
    rejected_at,
    expires_at,
    comment,
    reason
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
ON CONFLICT (confirmation_id) DO UPDATE SET
    deal_id = EXCLUDED.deal_id,
    stage = EXCLUDED.stage,
    requested_by_user_id = EXCLUDED.requested_by_user_id,
    requested_by_company_id = EXCLUDED.requested_by_company_id,
    counterparty_company_id = EXCLUDED.counterparty_company_id,
    status = EXCLUDED.status,
    verification_method = EXCLUDED.verification_method,
    verification_token_hash = EXCLUDED.verification_token_hash,
    signature_ref = EXCLUDED.signature_ref,
    requested_at = EXCLUDED.requested_at,
    approved_at = EXCLUDED.approved_at,
    rejected_at = EXCLUDED.rejected_at,
    expires_at = EXCLUDED.expires_at,
    comment = EXCLUDED.comment,
    reason = EXCLUDED.reason
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(
		ctx,
		query,
		item.ID(),
		item.DealID(),
		string(item.Stage()),
		item.RequestedByUserID(),
		item.RequestedByCompanyID(),
		item.CounterpartyCompanyID(),
		string(item.Status()),
		string(item.VerificationMethod()),
		nullString(item.VerificationTokenHash()),
		nullString(item.SignatureRef()),
		item.RequestedAt(),
		nullTime(item.ApprovedAt()),
		nullTime(item.RejectedAt()),
		nullTime(item.ExpiresAt()),
		nullString(item.Comment()),
		nullString(item.Reason()),
	)
	return err
}

func (r *DealConfirmationRepository) GetByID(ctx context.Context, confirmationID string) (*deal.DealConfirmation, error) {
	const query = `
SELECT confirmation_id, deal_id, stage, requested_by_user_id, requested_by_company_id, counterparty_company_id,
       status, verification_method, verification_token_hash, signature_ref, requested_at, approved_at, rejected_at,
       expires_at, comment, reason
FROM deal_confirmations
WHERE confirmation_id = $1
`
	return r.getOne(ctx, query, confirmationID)
}

func (r *DealConfirmationRepository) GetPendingByDealAndStage(
	ctx context.Context,
	dealID string,
	stage deal.DealConfirmationStage,
) (*deal.DealConfirmation, error) {
	const query = `
SELECT confirmation_id, deal_id, stage, requested_by_user_id, requested_by_company_id, counterparty_company_id,
       status, verification_method, verification_token_hash, signature_ref, requested_at, approved_at, rejected_at,
       expires_at, comment, reason
FROM deal_confirmations
WHERE deal_id = $1 AND stage = $2 AND status = 'pending'
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, dealID, string(stage))
	return scanDealConfirmation(row)
}

func (r *DealConfirmationRepository) ListByDealID(ctx context.Context, dealID string) ([]*deal.DealConfirmation, error) {
	const query = `
SELECT confirmation_id, deal_id, stage, requested_by_user_id, requested_by_company_id, counterparty_company_id,
       status, verification_method, verification_token_hash, signature_ref, requested_at, approved_at, rejected_at,
       expires_at, comment, reason
FROM deal_confirmations
WHERE deal_id = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	rows, err := dbtx.QueryContext(ctx, query, dealID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*deal.DealConfirmation, 0)
	for rows.Next() {
		item, err := scanDealConfirmation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	slices.SortFunc(items, func(left, right *deal.DealConfirmation) int {
		if left.RequestedAt().Equal(right.RequestedAt()) {
			switch {
			case left.ID() < right.ID():
				return -1
			case left.ID() > right.ID():
				return 1
			default:
				return 0
			}
		}
		if left.RequestedAt().After(right.RequestedAt()) {
			return -1
		}
		return 1
	})
	return items, nil
}

func (r *DealConfirmationRepository) getOne(ctx context.Context, query string, arg string) (*deal.DealConfirmation, error) {
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, arg)
	return scanDealConfirmation(row)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanDealConfirmation(row scanner) (*deal.DealConfirmation, error) {
	var (
		id                    string
		dealID                string
		stage                 string
		requestedByUserID     string
		requestedByCompanyID  string
		counterpartyCompanyID string
		status                string
		verificationMethod    string
		verificationTokenHash sql.NullString
		signatureRef          sql.NullString
		requestedAt           sql.NullTime
		approvedAt            sql.NullTime
		rejectedAt            sql.NullTime
		expiresAt             sql.NullTime
		comment               sql.NullString
		reason                sql.NullString
	)
	if err := row.Scan(
		&id,
		&dealID,
		&stage,
		&requestedByUserID,
		&requestedByCompanyID,
		&counterpartyCompanyID,
		&status,
		&verificationMethod,
		&verificationTokenHash,
		&signatureRef,
		&requestedAt,
		&approvedAt,
		&rejectedAt,
		&expiresAt,
		&comment,
		&reason,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, deal.ErrConfirmationNotFound
		}
		return nil, err
	}
	if !requestedAt.Valid {
		return nil, deal.ErrRequestedAtRequired
	}

	return deal.NewDealConfirmation(deal.DealConfirmationParams{
		ID:                    id,
		DealID:                dealID,
		Stage:                 deal.DealConfirmationStage(stage),
		RequestedByUserID:     requestedByUserID,
		RequestedByCompanyID:  requestedByCompanyID,
		CounterpartyCompanyID: counterpartyCompanyID,
		Status:                deal.DealConfirmationStatus(status),
		VerificationMethod:    deal.VerificationMethod(verificationMethod),
		VerificationTokenHash: verificationTokenHash.String,
		SignatureRef:          signatureRef.String,
		RequestedAt:           requestedAt.Time,
		ApprovedAt:            nullableTime(approvedAt),
		RejectedAt:            nullableTime(rejectedAt),
		ExpiresAt:             nullableTime(expiresAt),
		Comment:               comment.String,
		Reason:                reason.String,
	})
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
