package chainanchor

import (
	"context"
	"fmt"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/blockchain/evm"
	tradingpg "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/postgres"
)

type Worker struct {
	repo          *tradingpg.ChainOperationRepository
	client        *evm.RPCClient
	from          string
	contract      string
	submitBatch   int
	sentBatch     int
	retryDelay    time.Duration
	syncKey       string
	confirmations uint64
}

func NewWorker(
	repo *tradingpg.ChainOperationRepository,
	client *evm.RPCClient,
	from string,
	contract string,
) (*Worker, error) {
	if repo == nil {
		return nil, fmt.Errorf("chain operation repository is required")
	}
	if client == nil {
		return nil, fmt.Errorf("evm rpc client is required")
	}
	if from == "" {
		return nil, fmt.Errorf("from address is required")
	}
	if contract == "" {
		return nil, fmt.Errorf("contract address is required")
	}
	return &Worker{
		repo:          repo,
		client:        client,
		from:          from,
		contract:      contract,
		submitBatch:   100,
		sentBatch:     100,
		retryDelay:    10 * time.Second,
		syncKey:       "chain.logs.last_block",
		confirmations: 2,
	}, nil
}

func (w *Worker) SetConfirmations(confirmations uint64) {
	if confirmations == 0 {
		confirmations = 1
	}
	w.confirmations = confirmations
}

func (w *Worker) RunOnce(ctx context.Context) error {
	if err := w.submitPending(ctx); err != nil {
		return err
	}
	if err := w.reconcileSentReceipts(ctx); err != nil {
		return err
	}
	if err := w.syncEvents(ctx); err != nil {
		return err
	}
	return nil
}

func (w *Worker) submitPending(ctx context.Context) error {
	ops, err := w.repo.ListReadyForSubmit(ctx, time.Now().UTC(), w.submitBatch)
	if err != nil {
		return err
	}
	for _, op := range ops {
		var data string
		switch op.OpType {
		case tradingpg.ChainOpTypeAuctionCreate:
			if !op.StartsAt.Valid || !op.EndsAt.Valid || !op.MinBidStep.Valid {
				_ = w.repo.MarkSubmitFailed(ctx, op.OpID, "missing auction create payload", time.Now().UTC().Add(w.retryDelay))
				continue
			}
			data, err = evm.BuildCreateAuctionCallData(
				op.AuctionRefHash,
				op.MinBidStep.Int64,
				op.StartsAt.Time.Unix(),
				op.EndsAt.Time.Unix(),
				op.OpNonce,
			)
		case tradingpg.ChainOpTypeBidAnchor:
			if !op.BidHash.Valid || !op.PlacedAt.Valid {
				_ = w.repo.MarkSubmitFailed(ctx, op.OpID, "missing bid hash", time.Now().UTC().Add(w.retryDelay))
				continue
			}
			data, err = evm.BuildAnchorBidCallData(
				op.AuctionRefHash,
				op.BidHash.String,
				op.OpNonce,
				op.PlacedAt.Time.Unix(),
			)
		case tradingpg.ChainOpTypeAuctionFinalize:
			if !op.ResultHash.Valid || !op.WinnerCompanyID.Valid || !op.FinalPrice.Valid {
				_ = w.repo.MarkSubmitFailed(ctx, op.OpID, "missing finalize payload", time.Now().UTC().Add(w.retryDelay))
				continue
			}
			data, err = evm.BuildFinalizeAuctionCallData(
				op.AuctionRefHash,
				op.ResultHash.String,
				op.WinnerCompanyID.String,
				op.FinalPrice.Int64,
				op.OpNonce,
			)
		default:
			_ = w.repo.MarkSubmitFailed(ctx, op.OpID, "unknown op type", time.Now().UTC().Add(w.retryDelay))
			continue
		}
		if err != nil {
			_ = w.repo.MarkSubmitFailed(ctx, op.OpID, err.Error(), time.Now().UTC().Add(w.retryDelay))
			continue
		}
		txHash, err := w.client.SendTransaction(ctx, w.from, w.contract, data)
		if err != nil {
			_ = w.repo.MarkSubmitFailed(ctx, op.OpID, err.Error(), time.Now().UTC().Add(w.retryDelay))
			continue
		}
		if err := w.repo.MarkSubmitted(ctx, op.OpID, txHash, w.from); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) reconcileSentReceipts(ctx context.Context) error {
	ops, err := w.repo.ListSubmitted(ctx, w.sentBatch)
	if err != nil {
		return err
	}
	currentBlock, err := w.client.BlockNumber(ctx)
	if err != nil {
		return err
	}
	for _, op := range ops {
		if !op.TxHash.Valid || op.TxHash.String == "" {
			continue
		}
		receipt, err := w.client.GetTransactionReceipt(ctx, op.TxHash.String)
		if err != nil {
			_ = w.repo.MarkFailedByTxHash(ctx, op.TxHash.String, err.Error(), time.Now().UTC().Add(w.retryDelay))
			continue
		}
		if receipt == nil {
			continue
		}
		if receipt.Status == 1 {
			if !w.hasEnoughConfirmations(currentBlock, receipt.BlockNumber) {
				continue
			}
			if err := w.repo.MarkConfirmedByTxHash(ctx, op.TxHash.String, int64(receipt.BlockNumber)); err != nil {
				return err
			}
			continue
		}
		if err := w.repo.MarkFailedByTxHash(ctx, op.TxHash.String, "transaction reverted", time.Now().UTC().Add(w.retryDelay)); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) syncEvents(ctx context.Context) error {
	current, err := w.client.BlockNumber(ctx)
	if err != nil {
		return err
	}
	fromBlock, err := w.repo.GetSyncBlock(ctx, w.syncKey)
	if err != nil {
		return err
	}
	if fromBlock == 0 {
		fromBlock = current
	}
	if fromBlock > current {
		return w.repo.SetSyncBlock(ctx, w.syncKey, current)
	}
	if !w.hasEnoughConfirmations(current, fromBlock) {
		return nil
	}
	toBlock := current - (w.confirmations - 1)
	logs, err := w.client.GetLogs(ctx, fromBlock, toBlock, w.contract, []string{
		evm.AuctionCreatedTopic0(),
		evm.BidAnchoredTopic0(),
		evm.AuctionFinalizedTopic0(),
	})
	if err != nil {
		return err
	}
	for _, lg := range logs {
		if err := w.repo.MarkConfirmedByTxHash(ctx, lg.TxHash, int64(lg.BlockNumber)); err != nil {
			return err
		}
	}
	return w.repo.SetSyncBlock(ctx, w.syncKey, toBlock+1)
}

func (w *Worker) hasEnoughConfirmations(currentBlock, txBlock uint64) bool {
	if w.confirmations <= 1 {
		return currentBlock >= txBlock
	}
	if currentBlock < txBlock {
		return false
	}
	return (currentBlock - txBlock + 1) >= w.confirmations
}
