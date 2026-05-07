package app

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
)

var ErrNilDependency = errors.New("billing app: nil dependency")

func isBlank(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}

func auctionDepositLedgerRef(auctionID, companyID string) string {
	return "auction:" + auctionID + ":company:" + companyID
}

func platformFeeSettlementRef(auctionID, companyID string) string {
	return "auction:" + auctionID + ":company:" + companyID
}

func depositFromStartPrice(startPrice int64) int64 {
	d := startPrice * 5 / 100
	if d < 1 && startPrice > 0 {
		return 1
	}
	return d
}

func platformFeeFromFinalPrice(finalPrice int64) int64 {
	if finalPrice <= 0 {
		return 0
	}
	return finalPrice * 3 / 100
}

type RandomHexID struct{}

func (RandomHexID) NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
