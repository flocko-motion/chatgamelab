package functional

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateSecureToken generates a cryptographically secure random token.
// The token is URL-safe base64 encoded.
// byteLength specifies the number of random bytes (recommended: 32 for high security).
// The resulting token will be approximately byteLength * 4/3 characters long.
func GenerateSecureToken(byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	// Use URL-safe base64 encoding (no padding) for clean URLs
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
