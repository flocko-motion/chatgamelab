package db

import (
	"crypto/rand"
	"github.com/btcsuite/btcutil/base58"
)

// generateHash generates a random base58 string with at least 128 bits of entropy.
func generateHash() string {
	// 128 bits in binary is 16 bytes
	numBytes := 16

	randomBytes := make([]byte, numBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(err)
	}
	return base58.Encode(randomBytes)
}
