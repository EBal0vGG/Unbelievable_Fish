package app

import (
	"crypto/rand"
	"encoding/hex"
)

type RandomIDGenerator struct{}

func NewRandomIDGenerator() IDGenerator {
	return RandomIDGenerator{}
}

func (RandomIDGenerator) NewCompanyID() string {
	return newID("company")
}

func (RandomIDGenerator) NewUserID() string {
	return newID("user")
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
