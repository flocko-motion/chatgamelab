// package: auth / who is calling
// type:    logic
// job:     the ways in: an Auth0 identity mapped to its account, and a guest's login code redeemed
// limits:  schemaf verifies the Auth0 ID token and starts the session; no HTTP (-> api)
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cgl/db"
	"cgl/functional/wordtoken"
	"cgl/log"
	"cgl/tokenlock"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// auth0Provider is the name of schemaf's built-in Auth0 provider.
const auth0Provider = "auth0"

// OnLogin maps an identity schemaf has proven to the account it signs in:
// for Auth0, the account with that subject, created on the first sign-in with
// its registration incomplete. The person's details come from completing
// registration.
func OnLogin(ctx context.Context, id schemafapi.Identity) (string, error) {
	if id.Provider != auth0Provider {
		return "", fmt.Errorf("no accounts sign in with %q: %w", id.Provider, schemafapi.ErrForbidden)
	}
	userID, err := db.AccountForAuth0(ctx, id.Subject)
	if err != nil {
		return "", err
	}
	log.Debug("auth0 sign-in", "user_id", userID)
	return userID.String(), nil
}

var (
	// ErrInvalidGuestToken: no guest signs in with this code.
	ErrInvalidGuestToken = errors.New("invalid or expired token")
	// ErrNoGuestToken: the request names no token.
	ErrNoGuestToken = fmt.Errorf("a token is required: %w", schemafapi.ErrBadRequest)
)

// RedeemGuestToken returns the guest a login code signs in. The code comes
// whole (guest-…) or as its words, from a link, QR code or typed code. An
// unknown code counts towards the token lock.
//
// Codes issued before "guest-" start with db.LegacyGuestTokenPrefix
// ("participant-"): they stay valid, for backwards compatibility only, so
// typed words are tried with both prefixes.
func RedeemGuestToken(ctx context.Context, token string) (uuid.UUID, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return uuid.Nil, ErrNoGuestToken
	}
	candidates := []string{token}
	if !strings.HasPrefix(token, db.GuestTokenPrefix) && !strings.HasPrefix(token, db.LegacyGuestTokenPrefix) {
		words := wordtoken.Normalize(token)
		candidates = []string{
			db.GuestTokenPrefix + words,
			db.LegacyGuestTokenPrefix + words, // backwards compatibility only
		}
	}

	known := false
	for _, candidate := range candidates {
		user, err := db.GetUserByGuestToken(ctx, candidate)
		if err == nil {
			return user.ID, nil
		}
		var perr *db.GuestAuthError
		if errors.As(err, &perr) && perr.Code == "workshop_inactive" {
			return uuid.Nil, ErrWorkshopInactive
		}
		known = known || db.GuestTokenKnown(ctx, candidate)
	}
	if !known {
		tokenlock.Fail()
	}
	return uuid.Nil, ErrInvalidGuestToken
}
