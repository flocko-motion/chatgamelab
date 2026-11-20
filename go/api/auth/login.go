// package: auth / who is calling
// type:    logic
// job:     the ways in: an Auth0 identity mapped to its account, and a participant's token redeemed
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
	// ErrInvalidParticipantToken: no participant signs in with this token.
	ErrInvalidParticipantToken = errors.New("invalid or expired token")
	// ErrNoParticipantToken: the request names no token.
	ErrNoParticipantToken = fmt.Errorf("a token is required: %w", schemafapi.ErrBadRequest)
)

// RedeemParticipantToken returns the participant a token signs in. The token
// comes whole (participant-…) or as its words, from a link, QR code or typed
// code. An unknown token counts towards the token lock.
func RedeemParticipantToken(ctx context.Context, token string) (uuid.UUID, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return uuid.Nil, ErrNoParticipantToken
	}
	if !strings.HasPrefix(token, db.ParticipantTokenPrefix) {
		token = db.ParticipantTokenPrefix + wordtoken.Normalize(token)
	}

	user, err := db.GetUserByParticipantToken(ctx, token)
	if err != nil {
		if !db.ParticipantTokenKnown(ctx, token) {
			tokenlock.Fail()
		}
		var perr *db.ParticipantAuthError
		if errors.As(err, &perr) && perr.Code == "workshop_inactive" {
			return uuid.Nil, ErrWorkshopInactive
		}
		return uuid.Nil, ErrInvalidParticipantToken
	}
	return user.ID, nil
}
