package core

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"cgl/api/auth"
	"cgl/core/internal/store"
	"cgl/functional/wordtoken"
	"cgl/log"
	"cgl/tokenlock"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// The ways in: schemaf proves an identity, the core maps it to an account.
// An Auth0 subject signs in its account, made at its first sign-in with its
// registration incomplete; a workshop guest signs in with their login code.
// Signing in is also when an admin email becomes the first admin (while no
// admin exists): never while resolving a request's caller, which only reads.

// auth0Provider is the name of schemaf's built-in Auth0 provider.
const auth0Provider = "auth0"

// OnLogin is schemaf's sign-in hook: the account an Auth0 identity signs in.
//
//	app.OnLogin(core.OnLogin)
func OnLogin(ctx context.Context, id schemafapi.Identity) (string, error) {
	if id.Provider != auth0Provider {
		return "", fmt.Errorf("no accounts sign in with %q: %w", id.Provider, schemafapi.ErrForbidden)
	}
	account, err := auth0Account(ctx, id.Subject)
	if err != nil {
		return "", err
	}
	// an admin email counts from the sign-in; a new account has none yet and
	// gets its first role at registration
	if email, err := store.Queries().V2AccountEmail(ctx, account); err == nil && email.Valid {
		makeFirstAdmin(ctx, account, &email.String)
	}
	log.Debug("auth0 sign-in", "user_id", account)
	return account.String(), nil
}

// auth0Account is the account of an Auth0 subject, made at its first
// sign-in.
func auth0Account(ctx context.Context, subject string) (uuid.UUID, error) {
	sub := sql.NullString{String: subject, Valid: true}
	q := store.Queries()
	for range 2 { // a concurrent first sign-in may make it between the two
		ids, err := q.V2Auth0Account(ctx, sub)
		if err != nil || len(ids) > 0 {
			return first(ids), err
		}
		made, err := q.V2CreateAuth0Account(ctx, sub)
		if err != nil || len(made) > 0 {
			return first(made), err
		}
	}
	return uuid.Nil, fmt.Errorf("no account for subject %q", subject)
}

func first(ids []uuid.UUID) uuid.UUID {
	if len(ids) == 0 {
		return uuid.Nil
	}
	return ids[0]
}

// RedeemGuestToken is the guest a login code signs in. The code comes whole
// (guest-…) or as its words, from a link, a QR code or typed; codes begun
// with the legacy prefix stay valid. An unknown code counts towards the
// token lock; a guest whose workshop is not running is refused.
func RedeemGuestToken(ctx context.Context, token string) (uuid.UUID, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return uuid.Nil, auth.ErrNoGuestToken
	}
	candidates := []string{token}
	if !strings.HasPrefix(token, guestTokenPrefix) && !strings.HasPrefix(token, legacyGuestTokenPrefix) {
		words := wordtoken.Normalize(token)
		candidates = []string{guestTokenPrefix + words, legacyGuestTokenPrefix + words}
	}
	q := store.Queries()
	known := false
	for _, c := range candidates {
		code := sql.NullString{String: c, Valid: true}
		rows, err := q.V2GuestByToken(ctx, code)
		if err != nil {
			return uuid.Nil, err
		}
		if len(rows) > 0 {
			if !rows[0].WorkshopActive {
				return uuid.Nil, auth.ErrWorkshopInactive
			}
			return rows[0].ID, nil
		}
		// a code ever issued, of an account or workshop since gone, is no guess
		taken, err := q.V2GuestTokenExists(ctx, code)
		known = known || taken || err != nil
	}
	if !known {
		tokenlock.Fail()
	}
	return uuid.Nil, auth.ErrInvalidGuestToken
}

// DevSignInAccount is the account the dev sign-in signs in, in dev mode
// only (else ErrNotFound): a dev account by its id, or with uuid.Nil a new
// account still registering, as after a first Auth0 sign-in.
func DevSignInAccount(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	if !schemafapi.DevMode() {
		return uuid.Nil, ErrNotFound
	}
	if id == uuid.Nil {
		return auth0Account(ctx, "dev|"+uuid.NewString())
	}
	return id, nil
}
