package app

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/sha3"
)

type ChainOpsRepository interface {
	EnqueueAuctionCreate(ctx context.Context, in EnqueueAuctionCreateInput) error
	EnqueueBidAnchor(ctx context.Context, in EnqueueBidAnchorInput) error
	EnqueueAuctionFinalize(ctx context.Context, in EnqueueAuctionFinalizeInput) error
}

type EnqueueAuctionCreateInput struct {
	AuctionID      AuctionID
	AuctionRefHash string
	StartsAt       time.Time
	EndsAt         time.Time
	MinBidStep     int64
}

type EnqueueBidAnchorInput struct {
	AuctionID       AuctionID
	AuctionRefHash  string
	BidHash         string
	BidderCompanyID string
	Amount          int64
	PlacedAt        time.Time
}

type EnqueueAuctionFinalizeInput struct {
	AuctionID       AuctionID
	AuctionRefHash  string
	ResultHash      string
	WinnerCompanyID string
	FinalPrice      int64
}

type chainOpsTx interface {
	ChainOps() ChainOpsRepository
}

func buildAuctionRefHash(auctionID AuctionID) string {
	return keccakHex("auction|" + string(auctionID))
}

func buildBidHash(auctionID AuctionID, bidderCompanyID string, amount int64, placedAt time.Time) string {
	canonical := fmt.Sprintf("bid|%s|%s|%d|%s", string(auctionID), bidderCompanyID, amount, placedAt.UTC().Format(time.RFC3339Nano))
	return keccakHex(canonical)
}

func buildFinalizeResultHash(auctionID AuctionID, winnerCompanyID string, finalPrice int64) string {
	canonical := fmt.Sprintf("result|%s|%s|%d", string(auctionID), winnerCompanyID, finalPrice)
	return keccakHex(canonical)
}

func keccakHex(value string) string {
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(value))
	return "0x" + hex.EncodeToString(h.Sum(nil))
}

func normalizeHex32(value string) (string, error) {
	h := strings.ToLower(strings.TrimPrefix(value, "0x"))
	if len(h) != 64 {
		return "", fmt.Errorf("expected 32-byte hex value, got len=%d", len(h))
	}
	if _, err := hex.DecodeString(h); err != nil {
		return "", err
	}
	return "0x" + h, nil
}
