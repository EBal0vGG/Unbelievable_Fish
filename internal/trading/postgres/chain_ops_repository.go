package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

const (
	ChainOpTypeAuctionCreate   = "AUCTION_CREATE"
	ChainOpTypeBidAnchor       = "BID_ANCHOR"
	ChainOpTypeAuctionFinalize = "AUCTION_FINALIZE"

	ChainOpStatusCreated   = "CREATED"
	ChainOpStatusSent      = "SENT"
	ChainOpStatusConfirmed = "CONFIRMED"
	ChainOpStatusFailed    = "FAILED"

	BidChainStatusPending   = "PENDING_SUBMIT"
	BidChainStatusSent      = "SENT"
	BidChainStatusConfirmed = "CONFIRMED"
	BidChainStatusFailed    = "FAILED"

	AuctionChainStatusPending   = "PENDING_SUBMIT"
	AuctionChainStatusSent      = "SENT"
	AuctionChainStatusConfirmed = "CONFIRMED"
	AuctionChainStatusFailed    = "FAILED"
)

type ChainOperation struct {
	OpID            int64
	OpType          string
	OpNonce         int64
	AuctionID       string
	AuctionRefHash  string
	BidHash         sql.NullString
	PlacedAt        sql.NullTime
	StartsAt        sql.NullTime
	EndsAt          sql.NullTime
	MinBidStep      sql.NullInt64
	ResultHash      sql.NullString
	WinnerCompanyID sql.NullString
	FinalPrice      sql.NullInt64
	WalletAddress   sql.NullString
	Status          string
	TxHash          sql.NullString
	BlockNumber     sql.NullInt64
	LastError       sql.NullString
	AttemptCount    int
	NextRetryAt     time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ChainOperationRepository struct {
	db *sql.DB
}

func NewChainOperationRepository(db *sql.DB) *ChainOperationRepository {
	return &ChainOperationRepository{db: db}
}

var _ app.ChainOpsRepository = (*ChainOperationRepository)(nil)

func (r *ChainOperationRepository) EnqueueAuctionCreate(ctx context.Context, in app.EnqueueAuctionCreateInput) error {
	dbtx := DBTXFromContext(ctx, r.db)
	nonce, err := r.nextNonce(ctx, dbtx, string(in.AuctionID))
	if err != nil {
		return err
	}
	if _, err := dbtx.ExecContext(ctx, `
INSERT INTO trading_chain_operations (
    op_type,
    op_nonce,
    auction_id,
    auction_ref_hash,
    starts_at,
    ends_at,
    min_bid_step,
    status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, ChainOpTypeAuctionCreate, nonce, string(in.AuctionID), in.AuctionRefHash, in.StartsAt.UTC(), in.EndsAt.UTC(), in.MinBidStep, ChainOpStatusCreated); err != nil {
		return err
	}
	return nil
}

func (r *ChainOperationRepository) EnqueueBidAnchor(ctx context.Context, in app.EnqueueBidAnchorInput) error {
	dbtx := DBTXFromContext(ctx, r.db)
	nonce, err := r.nextNonce(ctx, dbtx, string(in.AuctionID))
	if err != nil {
		return err
	}
	if _, err := dbtx.ExecContext(ctx, `
INSERT INTO trading_chain_operations (
    op_type,
    op_nonce,
    auction_id,
    auction_ref_hash,
    bid_hash,
    placed_at,
    status
) VALUES ($1, $2, $3, $4, $5, $6, $7)
`, ChainOpTypeBidAnchor, nonce, string(in.AuctionID), in.AuctionRefHash, in.BidHash, in.PlacedAt.UTC(), ChainOpStatusCreated); err != nil {
		return err
	}

	res, err := dbtx.ExecContext(ctx, `
UPDATE trading_bids
SET chain_bid_hash = $1,
    chain_status = $2
WHERE auction_id = $3
  AND bidder_company_id = $4
  AND amount = $5
  AND placed_at = $6
`, in.BidHash, BidChainStatusPending, string(in.AuctionID), in.BidderCompanyID, in.Amount, in.PlacedAt)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("enqueue bid anchor: bid row not found for auction %s", in.AuctionID)
	}
	return nil
}

func (r *ChainOperationRepository) EnqueueAuctionFinalize(ctx context.Context, in app.EnqueueAuctionFinalizeInput) error {
	dbtx := DBTXFromContext(ctx, r.db)
	nonce, err := r.nextNonce(ctx, dbtx, string(in.AuctionID))
	if err != nil {
		return err
	}
	if _, err := dbtx.ExecContext(ctx, `
INSERT INTO trading_chain_operations (
    op_type,
    op_nonce,
    auction_id,
    auction_ref_hash,
    result_hash,
    winner_company_id,
    final_price,
    status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`, ChainOpTypeAuctionFinalize, nonce, string(in.AuctionID), in.AuctionRefHash, in.ResultHash, in.WinnerCompanyID, in.FinalPrice, ChainOpStatusCreated); err != nil {
		return err
	}

	res, err := dbtx.ExecContext(ctx, `
UPDATE trading_auctions
SET chain_result_hash = $1,
    chain_finalize_status = $2
WHERE auction_id = $3
`, in.ResultHash, AuctionChainStatusPending, string(in.AuctionID))
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("enqueue auction finalize: auction row not found for auction %s", in.AuctionID)
	}
	return nil
}

func (r *ChainOperationRepository) ListReadyForSubmit(ctx context.Context, now time.Time, limit int) ([]ChainOperation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, queryChainOps+`
WHERE status IN ($1, $2)
  AND next_retry_at <= $3
ORDER BY op_id ASC
LIMIT $4
`, ChainOpStatusCreated, ChainOpStatusFailed, now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChainOperations(rows)
}

func (r *ChainOperationRepository) ListSubmitted(ctx context.Context, limit int) ([]ChainOperation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, queryChainOps+`
WHERE status = $1
  AND tx_hash IS NOT NULL
ORDER BY op_id ASC
LIMIT $2
`, ChainOpStatusSent, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChainOperations(rows)
}

func (r *ChainOperationRepository) MarkSubmitted(ctx context.Context, opID int64, txHash, walletAddress string) error {
	var opType string
	var auctionID string
	var bidHash sql.NullString
	err := r.db.QueryRowContext(ctx, `
UPDATE trading_chain_operations
SET status = $1,
    tx_hash = $2,
    wallet_address = $3,
    last_error = NULL,
    attempt_count = attempt_count + 1,
    updated_at = NOW()
WHERE op_id = $4
RETURNING op_type, auction_id, bid_hash
`, ChainOpStatusSent, txHash, walletAddress, opID).Scan(&opType, &auctionID, &bidHash)
	if err != nil {
		return err
	}
	switch opType {
	case ChainOpTypeBidAnchor:
		if !bidHash.Valid {
			return fmt.Errorf("mark submitted: missing bid hash for op %d", opID)
		}
		_, err = r.db.ExecContext(ctx, `
UPDATE trading_bids
SET chain_tx_hash = $1,
    chain_status = $2,
    chain_wallet_address = $3
WHERE auction_id = $4
  AND chain_bid_hash = $5
`, txHash, BidChainStatusSent, walletAddress, auctionID, bidHash.String)
		return err
	case ChainOpTypeAuctionCreate:
		return nil
	case ChainOpTypeAuctionFinalize:
		_, err = r.db.ExecContext(ctx, `
UPDATE trading_auctions
SET chain_finalize_tx_hash = $1,
    chain_finalize_status = $2,
    chain_finalize_wallet_address = $3
WHERE auction_id = $4
`, txHash, AuctionChainStatusSent, walletAddress, auctionID)
		return err
	default:
		return fmt.Errorf("mark submitted: unknown op type %s", opType)
	}
}

func (r *ChainOperationRepository) MarkSubmitFailed(ctx context.Context, opID int64, lastError string, retryAt time.Time) error {
	var opType string
	var auctionID string
	var bidHash sql.NullString
	err := r.db.QueryRowContext(ctx, `
UPDATE trading_chain_operations
SET status = $1,
    last_error = $2,
    attempt_count = attempt_count + 1,
    next_retry_at = $3,
    updated_at = NOW()
WHERE op_id = $4
RETURNING op_type, auction_id, bid_hash
`, ChainOpStatusFailed, lastError, retryAt.UTC(), opID).Scan(&opType, &auctionID, &bidHash)
	if err != nil {
		return err
	}
	switch opType {
	case ChainOpTypeBidAnchor:
		if !bidHash.Valid {
			return fmt.Errorf("mark submit failed: missing bid hash for op %d", opID)
		}
		_, err = r.db.ExecContext(ctx, `
UPDATE trading_bids
SET chain_status = $1
WHERE auction_id = $2
  AND chain_bid_hash = $3
`, BidChainStatusFailed, auctionID, bidHash.String)
		return err
	case ChainOpTypeAuctionCreate:
		return nil
	case ChainOpTypeAuctionFinalize:
		_, err = r.db.ExecContext(ctx, `
UPDATE trading_auctions
SET chain_finalize_status = $1
WHERE auction_id = $2
`, AuctionChainStatusFailed, auctionID)
		return err
	default:
		return fmt.Errorf("mark submit failed: unknown op type %s", opType)
	}
}

func (r *ChainOperationRepository) MarkConfirmedByTxHash(ctx context.Context, txHash string, blockNumber int64) error {
	var opType string
	var auctionID string
	var bidHash sql.NullString
	err := r.db.QueryRowContext(ctx, `
UPDATE trading_chain_operations
SET status = $1,
    block_number = $2,
    updated_at = NOW()
WHERE tx_hash = $3
RETURNING op_type, auction_id, bid_hash
`, ChainOpStatusConfirmed, blockNumber, txHash).Scan(&opType, &auctionID, &bidHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	switch opType {
	case ChainOpTypeBidAnchor:
		if !bidHash.Valid {
			return fmt.Errorf("mark confirmed: missing bid hash for tx %s", txHash)
		}
		_, err = r.db.ExecContext(ctx, `
UPDATE trading_bids
SET chain_status = $1,
    chain_tx_hash = $2,
    chain_block_number = $3
WHERE auction_id = $4
  AND chain_bid_hash = $5
`, BidChainStatusConfirmed, txHash, blockNumber, auctionID, bidHash.String)
		return err
	case ChainOpTypeAuctionCreate:
		return nil
	case ChainOpTypeAuctionFinalize:
		_, err = r.db.ExecContext(ctx, `
UPDATE trading_auctions
SET chain_finalize_status = $1,
    chain_finalize_tx_hash = $2,
    chain_finalize_block_number = $3
WHERE auction_id = $4
`, AuctionChainStatusConfirmed, txHash, blockNumber, auctionID)
		return err
	default:
		return fmt.Errorf("mark confirmed: unknown op type %s", opType)
	}
}

func (r *ChainOperationRepository) MarkFailedByTxHash(ctx context.Context, txHash, lastError string, retryAt time.Time) error {
	var opType string
	var auctionID string
	var bidHash sql.NullString
	err := r.db.QueryRowContext(ctx, `
UPDATE trading_chain_operations
SET status = $1,
    last_error = $2,
    next_retry_at = $3,
    updated_at = NOW()
WHERE tx_hash = $4
RETURNING op_type, auction_id, bid_hash
`, ChainOpStatusFailed, lastError, retryAt.UTC(), txHash).Scan(&opType, &auctionID, &bidHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	switch opType {
	case ChainOpTypeBidAnchor:
		if !bidHash.Valid {
			return fmt.Errorf("mark failed by tx: missing bid hash for tx %s", txHash)
		}
		_, err = r.db.ExecContext(ctx, `
UPDATE trading_bids
SET chain_status = $1
WHERE auction_id = $2
  AND chain_bid_hash = $3
`, BidChainStatusFailed, auctionID, bidHash.String)
		return err
	case ChainOpTypeAuctionCreate:
		return nil
	case ChainOpTypeAuctionFinalize:
		_, err = r.db.ExecContext(ctx, `
UPDATE trading_auctions
SET chain_finalize_status = $1
WHERE auction_id = $2
`, AuctionChainStatusFailed, auctionID)
		return err
	default:
		return fmt.Errorf("mark failed by tx: unknown op type %s", opType)
	}
}

func (r *ChainOperationRepository) GetSyncBlock(ctx context.Context, key string) (uint64, error) {
	var value string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM trading_chain_sync_state WHERE key = $1`, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func (r *ChainOperationRepository) SetSyncBlock(ctx context.Context, key string, block uint64) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO trading_chain_sync_state (key, value, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW()
`, key, strconv.FormatUint(block, 10))
	return err
}

func (r *ChainOperationRepository) nextNonce(ctx context.Context, dbtx interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, auctionID string) (int64, error) {
	var nonce int64
	err := dbtx.QueryRowContext(ctx, `
SELECT COALESCE(MAX(op_nonce), -1) + 1
FROM trading_chain_operations
WHERE auction_id = $1
`, auctionID).Scan(&nonce)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

const queryChainOps = `
SELECT op_id, op_type, op_nonce, auction_id, auction_ref_hash, bid_hash, placed_at, starts_at, ends_at, min_bid_step,
       result_hash, winner_company_id, final_price, wallet_address,
       status, tx_hash, block_number, last_error, attempt_count, next_retry_at, created_at, updated_at
FROM trading_chain_operations
`

func scanChainOperations(rows *sql.Rows) ([]ChainOperation, error) {
	out := make([]ChainOperation, 0, 64)
	for rows.Next() {
		var item ChainOperation
		if err := rows.Scan(
			&item.OpID,
			&item.OpType,
			&item.OpNonce,
			&item.AuctionID,
			&item.AuctionRefHash,
			&item.BidHash,
			&item.PlacedAt,
			&item.StartsAt,
			&item.EndsAt,
			&item.MinBidStep,
			&item.ResultHash,
			&item.WinnerCompanyID,
			&item.FinalPrice,
			&item.WalletAddress,
			&item.Status,
			&item.TxHash,
			&item.BlockNumber,
			&item.LastError,
			&item.AttemptCount,
			&item.NextRetryAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
