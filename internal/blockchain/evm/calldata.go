package evm

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/sha3"
)

const (
	createAuctionSignature   = "createAuction(bytes32,uint256,uint64,uint64,uint64)"
	anchorBidSignature       = "anchorBid(bytes32,bytes32,uint64,uint64)"
	finalizeSignature        = "finalizeAuction(bytes32,bytes32,bytes32,uint256,uint64)"
	auctionCreatedEventSig   = "AuctionCreated(bytes32,uint256,uint64,uint64,uint64,address)"
	bidAnchoredEventSig      = "BidAnchored(bytes32,bytes32,uint64,uint64,address)"
	auctionFinalizedEventSig = "AuctionFinalized(bytes32,bytes32,bytes32,uint256,uint64,address)"
)

func AuctionCreatedTopic0() string { return signatureHash(auctionCreatedEventSig) }
func BidAnchoredTopic0() string    { return signatureHash(bidAnchoredEventSig) }
func AuctionFinalizedTopic0() string {
	return signatureHash(auctionFinalizedEventSig)
}

func BuildCreateAuctionCallData(auctionRefHash string, minBidStep int64, startsAtUnix, endsAtUnix int64, nonce int64) (string, error) {
	if minBidStep <= 0 {
		return "", fmt.Errorf("min bid step must be positive")
	}
	if startsAtUnix <= 0 || endsAtUnix <= 0 || startsAtUnix >= endsAtUnix {
		return "", fmt.Errorf("invalid starts/ends time")
	}
	if nonce < 0 {
		return "", fmt.Errorf("nonce must be non-negative")
	}
	selector := methodSelector(createAuctionSignature)
	auctionWord, err := asBytes32Word(auctionRefHash)
	if err != nil {
		return "", err
	}
	payload := make([]byte, 0, 4+32*5)
	payload = append(payload, selector[:]...)
	payload = append(payload, auctionWord[:]...)
	minBidStepWord := uint256Word(big.NewInt(minBidStep))
	startsAtWord := uint256Word(big.NewInt(startsAtUnix))
	endsAtWord := uint256Word(big.NewInt(endsAtUnix))
	nonceWord := uint256Word(big.NewInt(nonce))
	payload = append(payload, minBidStepWord[:]...)
	payload = append(payload, startsAtWord[:]...)
	payload = append(payload, endsAtWord[:]...)
	payload = append(payload, nonceWord[:]...)
	return "0x" + hex.EncodeToString(payload), nil
}

func BuildAnchorBidCallData(auctionRefHash, bidHash string, nonce int64, placedAtUnix int64) (string, error) {
	if nonce < 0 {
		return "", fmt.Errorf("nonce must be non-negative")
	}
	if placedAtUnix <= 0 {
		return "", fmt.Errorf("placedAt must be positive")
	}
	selector := methodSelector(anchorBidSignature)
	auctionWord, err := asBytes32Word(auctionRefHash)
	if err != nil {
		return "", err
	}
	bidWord, err := asBytes32Word(bidHash)
	if err != nil {
		return "", err
	}
	payload := make([]byte, 0, 4+32*4)
	payload = append(payload, selector[:]...)
	payload = append(payload, auctionWord[:]...)
	payload = append(payload, bidWord[:]...)
	nonceWord := uint256Word(big.NewInt(nonce))
	placedAtWord := uint256Word(big.NewInt(placedAtUnix))
	payload = append(payload, nonceWord[:]...)
	payload = append(payload, placedAtWord[:]...)
	return "0x" + hex.EncodeToString(payload), nil
}

func BuildFinalizeAuctionCallData(auctionRefHash, resultHash, winnerCompanyID string, finalPrice int64, nonce int64) (string, error) {
	if finalPrice <= 0 {
		return "", fmt.Errorf("final price must be positive")
	}
	if nonce < 0 {
		return "", fmt.Errorf("nonce must be non-negative")
	}
	selector := methodSelector(finalizeSignature)
	auctionWord, err := asBytes32Word(auctionRefHash)
	if err != nil {
		return "", err
	}
	resultWord, err := asBytes32Word(resultHash)
	if err != nil {
		return "", err
	}
	winnerWord := keccakWord("company|" + winnerCompanyID)
	payload := make([]byte, 0, 4+32*5)
	payload = append(payload, selector[:]...)
	payload = append(payload, auctionWord[:]...)
	payload = append(payload, resultWord[:]...)
	payload = append(payload, winnerWord[:]...)
	finalPriceWord := uint256Word(big.NewInt(finalPrice))
	nonceWord := uint256Word(big.NewInt(nonce))
	payload = append(payload, finalPriceWord[:]...)
	payload = append(payload, nonceWord[:]...)
	return "0x" + hex.EncodeToString(payload), nil
}

func signatureHash(signature string) string {
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(signature))
	return "0x" + hex.EncodeToString(h.Sum(nil))
}

func methodSelector(signature string) [4]byte {
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(signature))
	sum := h.Sum(nil)
	var out [4]byte
	copy(out[:], sum[:4])
	return out
}

func asBytes32Word(value string) ([32]byte, error) {
	var out [32]byte
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "0x")
	if len(normalized) != 64 {
		return out, fmt.Errorf("expected 32-byte hash hex, got len=%d", len(normalized))
	}
	decoded, err := hex.DecodeString(normalized)
	if err != nil {
		return out, err
	}
	copy(out[:], decoded)
	return out, nil
}

func keccakWord(value string) [32]byte {
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write([]byte(value))
	sum := h.Sum(nil)
	var out [32]byte
	copy(out[:], sum[:32])
	return out
}

func uint256Word(value *big.Int) [32]byte {
	var out [32]byte
	if value == nil {
		return out
	}
	b := value.Bytes()
	if len(b) > 32 {
		copy(out[:], b[len(b)-32:])
		return out
	}
	copy(out[32-len(b):], b)
	return out
}
