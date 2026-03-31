package app

import (
	"crypto/rand"
	"encoding/hex"
)

type IDGenerator interface {
	NewFishID() string
	NewProductID() string
	NewLotID() string
}

type RandomIDGenerator struct{}

func NewRandomIDGenerator() IDGenerator {
	return RandomIDGenerator{}
}

func (RandomIDGenerator) NewFishID() string {
	return newID("fish")
}

func (RandomIDGenerator) NewProductID() string {
	return newID("prod")
}

func (RandomIDGenerator) NewLotID() string {
	return newID("lot")
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
