package core

import (
	"context"
	"database/sql"
	"strings"

	"cgl/core/internal/store"
	"cgl/functional/wordtoken"
)

// A workshop guest signs in again with their login code: guestTokenPrefix
// and four words (44 bits); guessing is bounded by the token lock. Codes
// begun with legacyGuestTokenPrefix, from while guests were called
// participants, stay valid: printed links and QR codes carry them.
const (
	guestTokenPrefix       = "guest-"
	legacyGuestTokenPrefix = "participant-"
)

// newGuestToken is a login code no account has ever had.
func newGuestToken(ctx context.Context) (string, error) {
	words, err := wordtoken.GenerateUnique(ctx, 4, func(ctx context.Context, w string) (bool, error) {
		return store.Queries().V2GuestTokenExists(ctx, sql.NullString{String: guestTokenPrefix + w, Valid: true})
	})
	if err != nil {
		return "", err
	}
	return guestTokenPrefix + words, nil
}

// guestTokenWords are a login code's words, or "" for the long random codes
// issued before word codes.
func guestTokenWords(token string) string {
	words, ok := strings.CutPrefix(token, guestTokenPrefix)
	if !ok {
		words, ok = strings.CutPrefix(token, legacyGuestTokenPrefix)
	}
	if !ok || !wordtoken.IsWords(words) {
		return ""
	}
	return words
}
