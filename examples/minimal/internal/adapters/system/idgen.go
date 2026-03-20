package system

import (
	"crypto/rand"
	"encoding/hex"
)

// IDGenerator produces random 128-bit (hex-encoded) identifiers using crypto/rand.
// No external dependencies required — stdlib only.
type IDGenerator struct{}

func NewIDGenerator() IDGenerator { return IDGenerator{} }

func (IDGenerator) NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
