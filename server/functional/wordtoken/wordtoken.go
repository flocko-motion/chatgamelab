// package: functional / wordtoken
// type:    logic
// job:     generates hyphen-joined German word tokens that can be read aloud and typed
// limits:  token generation and input normalisation only; no storage or validation
package wordtoken

import (
	"context"
	"crypto/rand"
	_ "embed"
	"fmt"
	"math/big"
	"strings"
)

// 2048 words = 11 bits each. Derived from dys2p/wordlists-de de-2048-v1 (CC0); see README.md.
//
//go:embed words_de.txt
var wordsDE string

var words = strings.Fields(wordsDE)

const maxAttempts = 5

// Generate joins n uniformly random words with "-".
func Generate(n int) (string, error) {
	max := big.NewInt(int64(len(words)))
	parts := make([]string, n)
	for i := range parts {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("failed to generate word token: %w", err)
		}
		parts[i] = words[idx.Int64()]
	}
	return strings.Join(parts, "-"), nil
}

// GenerateUnique draws tokens until exists reports a free one.
// After maxAttempts collisions it retries once with n+1 words.
func GenerateUnique(ctx context.Context, n int, exists func(context.Context, string) (bool, error)) (string, error) {
	for attempt := 0; attempt <= maxAttempts; attempt++ {
		size := n
		if attempt == maxAttempts {
			size = n + 1
		}
		token, err := Generate(size)
		if err != nil {
			return "", err
		}
		taken, err := exists(ctx, token)
		if err != nil {
			return "", err
		}
		if !taken {
			return token, nil
		}
	}
	return "", fmt.Errorf("no free word token after %d attempts", maxAttempts+1)
}

var transliterate = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")

// Normalize maps typed input onto the stored form: lowercase, umlauts
// transliterated, and runs of whitespace, ".", "_" or "-" collapsed to one "-".
func Normalize(s string) string {
	s = transliterate.Replace(strings.ToLower(strings.TrimSpace(s)))
	fields := strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', '.', '_', '-':
			return true
		}
		return false
	})
	return strings.Join(fields, "-")
}
