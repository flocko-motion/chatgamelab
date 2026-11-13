// package: db / database access and repository layer
// type:    data
// job:     organisation self-registration: applications with e-mail tokens, confirmation that creates the institution, consent log, retention.
// limits:  no SQL (-> db/sqlc), no HTTP or mail (-> api/routes), declaration text comes from legal.
package db

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	sqlc "cgl/db/sqlc"
	"cgl/legal"
	"cgl/obj"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ApplicationTokenValidity is how long a confirmation link works.
const ApplicationTokenValidity = 48 * time.Hour

// field length limits for the registration form
const (
	maxPersonFieldLength      = 100
	maxInstitutionFieldLength = 150
	maxPostalCodeLength       = 20
)

// NormalizeInstitutionDetails trims all fields and validates them.
func NormalizeInstitutionDetails(d obj.InstitutionDetails) (obj.InstitutionDetails, *obj.AppError) {
	return normalizeInstitutionDetails(d, true)
}

// normalizeInstitutionDetails validates the form; withConstraint=false skips the
// youth-protection restriction (renewed consent keeps the organisation's current one).
func normalizeInstitutionDetails(d obj.InstitutionDetails, withConstraint bool) (obj.InstitutionDetails, *obj.AppError) {
	d.PersonName = strings.TrimSpace(d.PersonName)
	d.PersonFunction = strings.TrimSpace(d.PersonFunction)
	d.Email = strings.TrimSpace(d.Email)
	d.InstitutionName = strings.TrimSpace(d.InstitutionName)
	d.Operator = strings.TrimSpace(d.Operator)
	d.Street = strings.TrimSpace(d.Street)
	d.PostalCode = strings.TrimSpace(d.PostalCode)
	d.City = strings.TrimSpace(d.City)
	d.PromptConstraint = strings.TrimSpace(d.PromptConstraint)

	required := []struct {
		name, value string
		max         int
	}{
		{"personName", d.PersonName, maxPersonFieldLength},
		{"personFunction", d.PersonFunction, maxPersonFieldLength},
		{"email", d.Email, maxPersonFieldLength * 2},
		{"institutionName", d.InstitutionName, maxInstitutionFieldLength},
		{"operator", d.Operator, maxInstitutionFieldLength},
		{"street", d.Street, maxInstitutionFieldLength},
		{"postalCode", d.PostalCode, maxPostalCodeLength},
		{"city", d.City, maxInstitutionFieldLength},
		{"promptConstraint", d.PromptConstraint, obj.MaxPromptConstraintLength},
	}
	for _, f := range required {
		if f.name == "promptConstraint" && !withConstraint {
			continue
		}
		if f.value == "" {
			return d, obj.ErrValidationf("%s is required", f.name)
		}
		if utf8.RuneCountInString(f.value) > f.max {
			return d, obj.ErrValidationf("%s must be at most %d characters", f.name, f.max)
		}
	}
	addr, err := mail.ParseAddress(d.Email)
	if err != nil || addr.Address != d.Email {
		return d, obj.ErrValidation("invalid email address")
	}
	return d, nil
}

// CheckInstitutionEligibility says whether an account may register or upgrade to an organisation:
// only adults (u18) without any organisation, workshop or admin role.
func CheckInstitutionEligibility(user *obj.User) *obj.AppError {
	if user.AgeGroup == nil || *user.AgeGroup != obj.AgeGroupU18 {
		return obj.NewAppError(obj.ErrCodeInstitutionAgeNotAllowed, "only accounts of adults (18+) can register an organisation")
	}
	if user.Role != nil && user.Role.Role != obj.RoleIndividual {
		return obj.NewAppError(obj.ErrCodeAlreadyInInstitution, "account already has a role in an organisation, a workshop or as admin")
	}
	return nil
}

// checkEmailNotOtherAccount rejects an e-mail address that belongs to a different account.
// Knowing an address must never be enough to act for that account.
func checkEmailNotOtherAccount(ctx context.Context, userID uuid.UUID, email string) *obj.AppError {
	ids, err := queries().GetActiveUserIDsByEmail(ctx, email)
	if err != nil {
		return obj.ErrServerError("failed to check email")
	}
	for _, id := range ids {
		if id != userID {
			return obj.NewAppError(obj.ErrCodeEmailBelongsToOtherAccount, "this e-mail address belongs to another account; log in with that account to upgrade it")
		}
	}
	return nil
}

func checkInstitutionNameFree(ctx context.Context, name string) *obj.AppError {
	exists, err := queries().InstitutionNameExists(ctx, name)
	if err != nil {
		return obj.ErrServerError("failed to check organisation name")
	}
	if exists {
		return obj.NewAppError(obj.ErrCodeInstitutionNameTaken, "an organisation with this name already exists")
	}
	return nil
}

// PrecheckInstitutionDetails normalises the form and checks everything that does not need an
// account yet: field rules, that the e-mail belongs to no other account than accountID
// (uuid.Nil = no account yet), and that the organisation name is free.
func PrecheckInstitutionDetails(ctx context.Context, accountID uuid.UUID, d obj.InstitutionDetails) (obj.InstitutionDetails, *obj.AppError) {
	d, appErr := NormalizeInstitutionDetails(d)
	if appErr != nil {
		return d, appErr
	}
	if appErr := checkEmailNotOtherAccount(ctx, accountID, d.Email); appErr != nil {
		return d, appErr
	}
	if appErr := checkInstitutionNameFree(ctx, d.InstitutionName); appErr != nil {
		return d, appErr
	}
	return d, nil
}

// newApplicationToken returns a random URL-safe token and the hash stored for it.
func newApplicationToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashApplicationToken(token), nil
}

func hashApplicationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreateInstitutionApplication validates and stores a request. Older pending requests of the
// account are expired. Returns the application and the plain token for the confirmation link.
func CreateInstitutionApplication(ctx context.Context, userID uuid.UUID, kind string, d obj.InstitutionDetails) (*obj.InstitutionApplication, string, error) {
	if kind != obj.InstitutionApplicationRegister && kind != obj.InstitutionApplicationUpgrade {
		return nil, "", obj.ErrValidation("invalid application kind")
	}
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, "", obj.ErrNotFound("user not found")
	}
	if appErr := CheckInstitutionEligibility(user); appErr != nil {
		return nil, "", appErr
	}
	d, appErr := PrecheckInstitutionDetails(ctx, userID, d)
	if appErr != nil {
		return nil, "", appErr
	}

	token, hash, err := newApplicationToken()
	if err != nil {
		return nil, "", obj.ErrServerError("failed to create token")
	}
	if err := queries().ExpireInstitutionApplicationsForUser(ctx, uuid.NullUUID{UUID: userID, Valid: true}); err != nil {
		return nil, "", obj.ErrServerError("failed to expire older applications")
	}

	declaration := legal.CurrentDeclaration()
	row, err := queries().CreateInstitutionApplication(ctx, sqlc.CreateInstitutionApplicationParams{
		ID:                  uuid.New(),
		UserID:              uuid.NullUUID{UUID: userID, Valid: true},
		Kind:                kind,
		PersonName:          sql.NullString{String: d.PersonName, Valid: true},
		PersonFunction:      sql.NullString{String: d.PersonFunction, Valid: true},
		Email:               sql.NullString{String: d.Email, Valid: true},
		InstitutionName:     d.InstitutionName,
		InstitutionOperator: d.Operator,
		Street:              d.Street,
		PostalCode:          d.PostalCode,
		City:                d.City,
		PromptConstraint:    d.PromptConstraint,
		DeclarationVersion:  declaration.Version,
		DeclarationHash:     declaration.Hash,
		TokenHash:           hash,
		ExpiresAt:           time.Now().Add(ApplicationTokenValidity),
	})
	if err != nil {
		return nil, "", obj.ErrServerError("failed to store application")
	}
	return applicationFromRow(row), token, nil
}

// GetPendingInstitutionApplication returns the account's open request, or nil.
func GetPendingInstitutionApplication(ctx context.Context, userID uuid.UUID) (*obj.InstitutionApplication, error) {
	row, err := queries().GetPendingInstitutionApplicationByUser(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, obj.ErrServerError("failed to load application")
	}
	return applicationFromRow(row), nil
}

// RenewInstitutionApplicationToken gives the open request a fresh link (the old one stops working).
func RenewInstitutionApplicationToken(ctx context.Context, userID uuid.UUID) (*obj.InstitutionApplication, string, error) {
	app, err := GetPendingInstitutionApplication(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if app == nil {
		return nil, "", obj.ErrNotFound("no open application")
	}
	token, hash, err := newApplicationToken()
	if err != nil {
		return nil, "", obj.ErrServerError("failed to create token")
	}
	expires := time.Now().Add(ApplicationTokenValidity)
	if err := queries().ReplaceInstitutionApplicationToken(ctx, sqlc.ReplaceInstitutionApplicationTokenParams{
		ID: app.ID, TokenHash: hash, ExpiresAt: expires,
	}); err != nil {
		return nil, "", obj.ErrServerError("failed to renew link")
	}
	app.ExpiresAt = expires
	return app, token, nil
}

// InstitutionConfirmation is what a successful confirmation produced.
type InstitutionConfirmation struct {
	Institution *obj.Institution
	Application *obj.InstitutionApplication
	Consent     *obj.InstitutionConsent
}

// ConfirmInstitutionApplication turns a confirmed request into an organisation: in one transaction
// the institution is created with operator, address and youth-protection constraint, the account
// becomes its head, and the consent is logged.
func ConfirmInstitutionApplication(ctx context.Context, token string) (*InstitutionConfirmation, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, obj.NewAppError(obj.ErrCodeApplicationTokenInvalid, "missing token")
	}
	row, err := queries().GetInstitutionApplicationByTokenHash(ctx, hashApplicationToken(token))
	if err != nil {
		return nil, obj.NewAppError(obj.ErrCodeApplicationTokenInvalid, "unknown confirmation link")
	}
	if row.Status == obj.InstitutionApplicationConfirmed {
		return nil, obj.NewAppError(obj.ErrCodeApplicationTokenInvalid, "this link was already used")
	}
	if row.Status == obj.InstitutionApplicationExpired || time.Now().After(row.ExpiresAt) {
		return nil, obj.NewAppError(obj.ErrCodeApplicationExpired, "this link has expired")
	}
	if !row.UserID.Valid || !row.PersonName.Valid || !row.Email.Valid {
		return nil, obj.NewAppError(obj.ErrCodeApplicationTokenInvalid, "application is no longer valid")
	}
	declaration, ok := legal.DeclarationByVersion(row.DeclarationVersion)
	if !ok || declaration.Hash != row.DeclarationHash || declaration.Version != legal.CurrentDeclaration().Version {
		return nil, obj.NewAppError(obj.ErrCodeDeclarationVersionOutdated, "the declaration changed since the application; please apply again")
	}

	userID := row.UserID.UUID
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.ErrNotFound("account not found")
	}
	if appErr := CheckInstitutionEligibility(user); appErr != nil {
		return nil, appErr
	}
	if appErr := checkInstitutionNameFree(ctx, row.InstitutionName); appErr != nil {
		return nil, appErr
	}

	tx, err := sqlDb.BeginTx(ctx, nil)
	if err != nil {
		return nil, obj.ErrServerError("failed to begin transaction")
	}
	defer tx.Rollback()
	q := queries().WithTx(tx)

	now := time.Now()
	creator := uuid.NullUUID{UUID: userID, Valid: true}
	inst, err := q.CreateInstitution(ctx, sqlc.CreateInstitutionParams{
		ID: uuid.New(), CreatedBy: creator, CreatedAt: now, ModifiedBy: creator, ModifiedAt: now,
		Name: row.InstitutionName,
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, obj.NewAppError(obj.ErrCodeInstitutionNameTaken, "an organisation with this name already exists")
		}
		return nil, obj.ErrServerError("failed to create organisation")
	}
	if err := q.UpdateInstitutionDetails(ctx, sqlc.UpdateInstitutionDetailsParams{
		ID:         inst.ID,
		Operator:   sql.NullString{String: row.InstitutionOperator, Valid: true},
		Street:     sql.NullString{String: row.Street, Valid: true},
		PostalCode: sql.NullString{String: row.PostalCode, Valid: true},
		City:       sql.NullString{String: row.City, Valid: true},
	}); err != nil {
		return nil, obj.ErrServerError("failed to store organisation details")
	}
	if err := q.UpdateInstitutionPromptConstraints(ctx, sqlc.UpdateInstitutionPromptConstraintsParams{
		ID: inst.ID, PromptConstraints: sql.NullString{String: row.PromptConstraint, Valid: true},
	}); err != nil {
		return nil, obj.ErrServerError("failed to store youth-protection restriction")
	}
	if err := q.DeleteUserRoles(ctx, userID); err != nil {
		return nil, obj.ErrServerError("failed to replace role")
	}
	if _, err := q.CreateUserRole(ctx, sqlc.CreateUserRoleParams{
		UserID:        userID,
		Role:          sql.NullString{String: string(obj.RoleHead), Valid: true},
		InstitutionID: uuid.NullUUID{UUID: inst.ID, Valid: true},
	}); err != nil {
		return nil, obj.ErrServerError("failed to assign head role")
	}
	consentRow, err := q.CreateInstitutionConsent(ctx, sqlc.CreateInstitutionConsentParams{
		ID:                  uuid.New(),
		InstitutionID:       inst.ID,
		UserID:              userID,
		ApplicationID:       uuid.NullUUID{UUID: row.ID, Valid: true},
		Email:               row.Email.String,
		PersonName:          row.PersonName.String,
		PersonFunction:      row.PersonFunction.String,
		InstitutionName:     row.InstitutionName,
		InstitutionOperator: row.InstitutionOperator,
		Street:              row.Street,
		PostalCode:          row.PostalCode,
		City:                row.City,
		DeclarationVersion:  declaration.Version,
		DeclarationHash:     declaration.Hash,
	})
	if err != nil {
		return nil, obj.ErrServerError("failed to log consent")
	}
	if err := q.ConfirmInstitutionApplication(ctx, sqlc.ConfirmInstitutionApplicationParams{
		ID: row.ID, InstitutionID: uuid.NullUUID{UUID: inst.ID, Valid: true},
	}); err != nil {
		return nil, obj.ErrServerError("failed to confirm application")
	}
	if err := tx.Commit(); err != nil {
		return nil, obj.ErrServerError("failed to commit organisation registration")
	}

	app := applicationFromRow(row)
	app.Status = obj.InstitutionApplicationConfirmed
	app.InstitutionID = &inst.ID
	constraint := row.PromptConstraint
	return &InstitutionConfirmation{
		Institution: &obj.Institution{
			ID: inst.ID, Name: inst.Name, PromptConstraints: &constraint,
			Meta: obj.Meta{CreatedBy: inst.CreatedBy, CreatedAt: &inst.CreatedAt, ModifiedBy: inst.ModifiedBy, ModifiedAt: &inst.ModifiedAt},
		},
		Application: app,
		Consent:     consentFromRow(consentRow),
	}, nil
}

// GetLatestInstitutionConsent returns the newest consent of an organisation, or nil.
func GetLatestInstitutionConsent(ctx context.Context, institutionID uuid.UUID) (*obj.InstitutionConsent, error) {
	row, err := queries().GetLatestInstitutionConsent(ctx, institutionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, obj.ErrServerError("failed to load consent")
	}
	return consentFromRow(row), nil
}

// ExpireAndAnonymizeInstitutionApplications marks overdue requests expired and removes the
// personal data of unconfirmed requests older than the retention window.
func ExpireAndAnonymizeInstitutionApplications(ctx context.Context, retention time.Duration) (expired, anonymized int64, err error) {
	expired, err = queries().ExpireOverdueInstitutionApplications(ctx)
	if err != nil {
		return 0, 0, err
	}
	anonymized, err = queries().AnonymizeOldUnconfirmedInstitutionApplications(ctx, time.Now().Add(-retention))
	return expired, anonymized, err
}

func applicationFromRow(r sqlc.InstitutionApplication) *obj.InstitutionApplication {
	a := &obj.InstitutionApplication{
		ID:                 r.ID,
		CreatedAt:          r.CreatedAt,
		UserID:             nullUUIDToPtr(r.UserID),
		Kind:               r.Kind,
		PersonName:         nullStringToPtr(r.PersonName),
		PersonFunction:     nullStringToPtr(r.PersonFunction),
		Email:              nullStringToPtr(r.Email),
		InstitutionName:    r.InstitutionName,
		Operator:           r.InstitutionOperator,
		Street:             r.Street,
		PostalCode:         r.PostalCode,
		City:               r.City,
		PromptConstraint:   r.PromptConstraint,
		DeclarationVersion: r.DeclarationVersion,
		ExpiresAt:          r.ExpiresAt,
		Status:             r.Status,
		InstitutionID:      nullUUIDToPtr(r.InstitutionID),
	}
	if r.ConfirmedAt.Valid {
		a.ConfirmedAt = &r.ConfirmedAt.Time
	}
	if r.AnonymizedAt.Valid {
		a.AnonymizedAt = &r.AnonymizedAt.Time
	}
	return a
}

func consentFromRow(r sqlc.InstitutionConsent) *obj.InstitutionConsent {
	return &obj.InstitutionConsent{
		ID:                 r.ID,
		CreatedAt:          r.CreatedAt,
		InstitutionID:      r.InstitutionID,
		UserID:             r.UserID,
		ApplicationID:      nullUUIDToPtr(r.ApplicationID),
		Email:              r.Email,
		PersonName:         r.PersonName,
		PersonFunction:     r.PersonFunction,
		InstitutionName:    r.InstitutionName,
		Operator:           r.InstitutionOperator,
		Street:             r.Street,
		PostalCode:         r.PostalCode,
		City:               r.City,
		DeclarationVersion: r.DeclarationVersion,
		DeclarationHash:    r.DeclarationHash,
	}
}

// InstitutionDeclarationStatusFor compares the organisation's latest consent with the current declaration.
func InstitutionDeclarationStatusFor(ctx context.Context, institutionID uuid.UUID) (*obj.InstitutionDeclarationStatus, error) {
	current := legal.CurrentDeclaration()
	status := &obj.InstitutionDeclarationStatus{CurrentVersion: current.Version, Pending: true}
	latest, err := GetLatestInstitutionConsent(ctx, institutionID)
	if err != nil {
		return nil, err
	}
	if latest != nil {
		status.ConsentedVersion = &latest.DeclarationVersion
		status.ConsentedAt = &latest.CreatedAt
		status.Pending = latest.DeclarationVersion != current.Version || latest.DeclarationHash != current.Hash
	}
	return status, nil
}

// InstitutionConsentInput is what a head enters when consenting for an existing organisation.
type InstitutionConsentInput struct {
	PersonName     string
	PersonFunction string
	Email          string
	Operator       string
	Street         string
	PostalCode     string
	City           string
}

// RecordInstitutionConsent logs a head's consent to the current declaration for their own
// organisation and stores operator and address. One consent per organisation and version is enough;
// a further consent simply adds another record.
func RecordInstitutionConsent(ctx context.Context, userID, institutionID uuid.UUID, in InstitutionConsentInput) (*obj.InstitutionConsent, error) {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return nil, obj.ErrNotFound("user not found")
	}
	if user.Role == nil || user.Role.Role != obj.RoleHead || user.Role.Institution == nil || user.Role.Institution.ID != institutionID {
		return nil, obj.ErrForbidden("only a head of this organisation can consent for it")
	}
	inst, err := queries().GetInstitutionByID(ctx, institutionID)
	if err != nil {
		return nil, obj.ErrNotFound("organisation not found")
	}

	// Reuse the registration rules; name and restriction come from the organisation itself.
	d, appErr := normalizeInstitutionDetails(obj.InstitutionDetails{
		PersonName: in.PersonName, PersonFunction: in.PersonFunction, Email: in.Email,
		InstitutionName: inst.Name, Operator: in.Operator, Street: in.Street,
		PostalCode: in.PostalCode, City: in.City,
	}, false)
	if appErr != nil {
		return nil, appErr
	}

	declaration := legal.CurrentDeclaration()
	tx, err := sqlDb.BeginTx(ctx, nil)
	if err != nil {
		return nil, obj.ErrServerError("failed to begin transaction")
	}
	defer tx.Rollback()
	q := queries().WithTx(tx)

	if err := q.UpdateInstitutionDetails(ctx, sqlc.UpdateInstitutionDetailsParams{
		ID:         institutionID,
		Operator:   sql.NullString{String: d.Operator, Valid: true},
		Street:     sql.NullString{String: d.Street, Valid: true},
		PostalCode: sql.NullString{String: d.PostalCode, Valid: true},
		City:       sql.NullString{String: d.City, Valid: true},
	}); err != nil {
		return nil, obj.ErrServerError("failed to store organisation details")
	}
	row, err := q.CreateInstitutionConsent(ctx, sqlc.CreateInstitutionConsentParams{
		ID:                  uuid.New(),
		InstitutionID:       institutionID,
		UserID:              userID,
		Email:               d.Email,
		PersonName:          d.PersonName,
		PersonFunction:      d.PersonFunction,
		InstitutionName:     inst.Name,
		InstitutionOperator: d.Operator,
		Street:              d.Street,
		PostalCode:          d.PostalCode,
		City:                d.City,
		DeclarationVersion:  declaration.Version,
		DeclarationHash:     declaration.Hash,
	})
	if err != nil {
		return nil, obj.ErrServerError("failed to log consent")
	}
	if err := tx.Commit(); err != nil {
		return nil, obj.ErrServerError("failed to commit consent")
	}
	return consentFromRow(row), nil
}
