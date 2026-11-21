package core

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"cgl/core/internal/identity"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/log"

	"github.com/google/uuid"
)

// An organisation is suspended while an admin suspended it, or while it has
// no consent to the current declaration (organisation_consent.go). Its
// heads, staff and the guests in its workshops are refused until the
// suspension ends (the Suspended context), its share links and public pages
// stop, and nothing is lost. An admin suspends instead of deleting; only an
// admin lifts their suspension.

// OrganisationSuspension is an admin's suspension of an organisation: when
// and why, both nil while there is none.
type OrganisationSuspension struct {
	ID          uuid.UUID
	SuspendedAt *time.Time
	Reason      *string
}

// SuspendOrganisation suspends organisation id, with an optional note on
// why, and returns the suspension. It takes {Suspend, OrganisationCard}.
func SuspendOrganisation(ctx context.Context, id uuid.UUID, reason string) (OrganisationSuspension, error) {
	now := time.Now()
	s := OrganisationSuspension{ID: id, SuspendedAt: &now}
	if reason = strings.TrimSpace(reason); reason != "" {
		s.Reason = &reason
	}
	return s, setSuspension(ctx, s)
}

// LiftSuspension lifts an admin's suspension of organisation id. It takes
// {Suspend, OrganisationCard}; a suspension for want of a consent stays
// until a head consents.
func LiftSuspension(ctx context.Context, id uuid.UUID) (OrganisationSuspension, error) {
	s := OrganisationSuspension{ID: id}
	return s, setSuspension(ctx, s)
}

// setSuspension stores an admin's suspension s, for a caller who may
// suspend the organisation. One the caller may not read does not exist for
// them (ErrNotFound).
func setSuspension(ctx context.Context, s OrganisationSuspension) error {
	org, err := ReadOrganisation(ctx, s.ID)
	if err != nil {
		return err
	}
	if !org.MaySuspend() {
		return forbidden("only an admin suspends an organisation")
	}
	params := db.V2SetOrganisationSuspensionParams{ID: s.ID, ModifiedBy: account(ctx).UUID}
	if s.SuspendedAt != nil {
		params.SuspendedAt = sql.NullTime{Time: *s.SuspendedAt, Valid: true}
	}
	params.SuspendedReason = sqlString(s.Reason)
	n, err := store.Queries().V2SetOrganisationSuspension(ctx, params)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	log.Info("organisation suspension changed", "institution_id", s.ID, "suspended", s.SuspendedAt != nil, "admin_id", account(ctx).UUID)
	return nil
}

// Why the caller's organisation is suspended, as ReadMyOrganisation reports it.
const (
	SuspendedByAdmin   = string(identity.SuspendedByAdmin)   // only an admin lifts it
	SuspendedNoConsent = string(identity.SuspendedNoConsent) // a head lifts it by consenting
)

// MyOrganisation is what the caller's own view adds about organisations:
// why theirs is suspended (SuspendedByAdmin, SuspendedNoConsent; empty while
// it is not), whether they may consent in its name, and whether they may
// suspend organisations.
type MyOrganisation struct {
	Suspension string
	MayConsent bool
	MaySuspend bool
}

// ReadMyOrganisation is the caller's MyOrganisation.
func ReadMyOrganisation(ctx context.Context) MyOrganisation {
	return MyOrganisation{
		Suspension: string(from(ctx).Suspension()),
		MayConsent: mayConsentForOwnOrganisation(ctx),
		MaySuspend: maySuspendOrganisations(ctx),
	}
}
