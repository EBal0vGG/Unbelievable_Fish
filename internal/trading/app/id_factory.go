package app

import (
	"crypto/rand"
	"encoding/hex"
)

type AuctionIDFactory interface {
	NewID() (AuctionID, error)
}

type RandomAuctionIDFactory struct{}

func (RandomAuctionIDFactory) NewID() (AuctionID, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return AuctionID(hex.EncodeToString(buf)), nil
}
