// package: db / database access and repository layer
// type:    data
// job:     admin side of organisations: reversible lock, lock checks for members and shares, list of registration applications and fresh confirmation links for them, details on manual creation.
// limits:  no SQL (-> db/sqlc), no HTTP (-> api/routes/institution_admin.go).
package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	sqlc "cgl/db/sqlc"
	"cgl/obj"

	"github.com/google/uuid"
)

// maxApplicationsListed caps the admin list of registration applications.
const maxApplicationsListed = 500

func requireAdmin(ctx context.Context, userID uuid.UUID) error {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return obj.ErrNotFound("user not found")
	}
	if user.Role == nil || user.Role.Role != obj.RoleAdmin {
		return obj.ErrForbidden("admin only")
	}
	return nil
}

// SetInstitutionLock locks (disabled=true) or unlocks an organisation. Admin only. Data is kept.
func SetInstitutionLock(ctx context.Context, adminID, institutionID uuid.UUID, disabled bool, reason string) error {
	if err := requireAdmin(ctx, adminID); err != nil {
		return err
	}
	if _, err := queries().GetInstitutionByID(ctx, institutionID); err != nil {
		return obj.ErrNotFound("organisation not found")
	}
	params := sqlc.SetInstitutionDisabledParams{ID: institutionID}
	if disabled {
		params.DisabledAt = sql.NullTime{Time: time.Now(), Valid: true}
		reason = strings.TrimSpace(reason)
		params.DisabledReason = sql.NullString{String: reason, Valid: reason != ""}
	}
	if err := queries().SetInstitutionDisabled(ctx, params); err != nil {
		return obj.ErrServerError("failed to change lock")
	}
	return nil
}

// IsUserInstitutionDisabled reports whether the user is head, staff or participant of a locked organisation.
func IsUserInstitutionDisabled(ctx context.Context, userID uuid.UUID) bool {
	disabled, err := queries().IsUserInstitutionDisabled(ctx, userID)
	return err == nil && disabled
}

// IsShareInstitutionDisabled reports whether a game share belongs to a locked organisation.
func IsShareInstitutionDisabled(ctx context.Context, share *obj.GameShare) bool {
	if share == nil || (share.InstitutionID == nil && share.WorkshopID == nil) {
		return false
	}
	params := sqlc.IsShareInstitutionDisabledParams{}
	if share.InstitutionID != nil {
		params.InstitutionID = uuid.NullUUID{UUID: *share.InstitutionID, Valid: true}
	}
	if share.WorkshopID != nil {
		params.WorkshopID = uuid.NullUUID{UUID: *share.WorkshopID, Valid: true}
	}
	disabled, err := queries().IsShareInstitutionDisabled(ctx, params)
	return err == nil && disabled
}

// ListInstitutionApplications returns all registration applications, newest first. Admin only.
func ListInstitutionApplications(ctx context.Context, adminID uuid.UUID) ([]obj.InstitutionApplication, error) {
	if err := requireAdmin(ctx, adminID); err != nil {
		return nil, err
	}
	rows, err := queries().ListInstitutionApplications(ctx, maxApplicationsListed)
	if err != nil {
		return nil, obj.ErrServerError("failed to list applications")
	}
	out := make([]obj.InstitutionApplication, 0, len(rows))
	for _, r := range rows {
		a := applicationFromRow(r)
		// An overdue application counts as expired even before the daily job marks it.
		if a.Status == obj.InstitutionApplicationPending && time.Now().After(a.ExpiresAt) {
			a.Status = obj.InstitutionApplicationExpired
		}
		out = append(out, *a)
	}
	return out, nil
}

// SetInstitutionDetails stores operator and address (admin creating an organisation by hand).
func SetInstitutionDetails(ctx context.Context, institutionID uuid.UUID, operator, street, postalCode, city string) error {
	ns := func(s string) sql.NullString {
		s = strings.TrimSpace(s)
		return sql.NullString{String: s, Valid: s != ""}
	}
	return queries().UpdateInstitutionDetails(ctx, sqlc.UpdateInstitutionDetailsParams{
		ID: institutionID, Operator: ns(operator), Street: ns(street), PostalCode: ns(postalCode), City: ns(city),
	})
}

// RenewInstitutionApplicationLinkAsAdmin gives an open or lapsed application a fresh confirmation
// link, so an admin can send it by hand (e.g. while no mail server is configured). The previous
// link stops working. Admin only. Returns the application and the plain token.
func RenewInstitutionApplicationLinkAsAdmin(ctx context.Context, adminID, applicationID uuid.UUID) (*obj.InstitutionApplication, string, error) {
	if err := requireAdmin(ctx, adminID); err != nil {
		return nil, "", err
	}
	row, err := queries().GetInstitutionApplicationByID(ctx, applicationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", obj.ErrNotFound("application not found")
	}
	if err != nil {
		return nil, "", obj.ErrServerError("failed to load application")
	}
	app := applicationFromRow(row)
	switch {
	case app.Status == obj.InstitutionApplicationConfirmed:
		return nil, "", obj.ErrConflict("application is already confirmed")
	case app.AnonymizedAt != nil || app.UserID == nil || app.Email == nil:
		return nil, "", obj.ErrConflict("application was anonymised")
	}
	if app.Status == obj.InstitutionApplicationExpired {
		// Only revive it if the person has not applied again in the meantime.
		pending, err := GetPendingInstitutionApplication(ctx, *app.UserID)
		if err != nil {
			return nil, "", err
		}
		if pending != nil && pending.ID != app.ID {
			return nil, "", obj.ErrConflict("the person has a newer application")
		}
	}

	token, hash, err := newApplicationToken()
	if err != nil {
		return nil, "", obj.ErrServerError("failed to create token")
	}
	expires := time.Now().Add(ApplicationTokenValidity)
	n, err := queries().RenewInstitutionApplicationTokenByID(ctx, sqlc.RenewInstitutionApplicationTokenByIDParams{
		ID: app.ID, TokenHash: hash, ExpiresAt: expires,
	})
	if err != nil {
		return nil, "", obj.ErrServerError("failed to renew link")
	}
	if n == 0 {
		return nil, "", obj.ErrConflict("application can no longer be renewed")
	}
	app.ExpiresAt = expires
	app.Status = obj.InstitutionApplicationPending
	return app, token, nil
}
