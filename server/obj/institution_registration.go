// package: obj / organisation self-registration types
// type:    data
// job:     defines the application, consent and form types plus the error codes of the organisation registration flow.
// limits:  no persistence (-> db/institution_registration.go), no HTTP (-> api/routes/institution_registration.go).
package obj

import (
	"time"

	"github.com/google/uuid"
)

// MaxPromptConstraintLength is the maximum length (in characters) of an organisation's
// Jugendschutz-Prompt-Einschränkung. Kept short on purpose.
const MaxPromptConstraintLength = 200

// Institution application kinds.
const (
	InstitutionApplicationRegister = "register" // new account registers as an organisation
	InstitutionApplicationUpgrade  = "upgrade"  // existing account upgrades to an organisation
)

// Institution application states.
const (
	InstitutionApplicationPending   = "pending"
	InstitutionApplicationConfirmed = "confirmed"
	InstitutionApplicationExpired   = "expired"
)

// Error codes of the registration flow (the web app maps them to texts).
const (
	ErrCodeInstitutionAgeNotAllowed    = "institution_age_not_allowed"
	ErrCodeAlreadyInInstitution        = "already_in_institution"
	ErrCodeInstitutionNameTaken        = "institution_name_taken"
	ErrCodeEmailBelongsToOtherAccount  = "email_belongs_to_other_account"
	ErrCodeApplicationTokenInvalid     = "application_token_invalid"
	ErrCodeApplicationExpired          = "application_expired"
	ErrCodeDeclarationVersionOutdated  = "declaration_version_outdated"
	ErrCodeInstitutionConsentsRequired = "institution_consents_required"
)

// InstitutionDetails is what a person enters about the organisation and themselves.
type InstitutionDetails struct {
	PersonName       string `json:"personName"`
	PersonFunction   string `json:"personFunction"`
	Email            string `json:"email"`
	InstitutionName  string `json:"institutionName"`
	Operator         string `json:"operator"`
	Street           string `json:"street"`
	PostalCode       string `json:"postalCode"`
	City             string `json:"city"`
	PromptConstraint string `json:"promptConstraint"`
}

// InstitutionApplication is a registration/upgrade request waiting for e-mail confirmation.
// Personal fields are nil once the application was anonymised.
type InstitutionApplication struct {
	ID                 uuid.UUID  `json:"id"`
	CreatedAt          time.Time  `json:"createdAt"`
	UserID             *uuid.UUID `json:"userId,omitempty"`
	Kind               string     `json:"kind"`
	PersonName         *string    `json:"personName,omitempty"`
	PersonFunction     *string    `json:"personFunction,omitempty"`
	Email              *string    `json:"email,omitempty"`
	InstitutionName    string     `json:"institutionName"`
	Operator           string     `json:"operator"`
	Street             string     `json:"street"`
	PostalCode         string     `json:"postalCode"`
	City               string     `json:"city"`
	PromptConstraint   string     `json:"promptConstraint"`
	DeclarationVersion string     `json:"declarationVersion"`
	ExpiresAt          time.Time  `json:"expiresAt"`
	Status             string     `json:"status"`
	ConfirmedAt        *time.Time `json:"confirmedAt,omitempty"`
	InstitutionID      *uuid.UUID `json:"institutionId,omitempty"`
	AnonymizedAt       *time.Time `json:"anonymizedAt,omitempty"`
}

// InstitutionConsent is one immutable record of a declaration consent.
type InstitutionConsent struct {
	ID                 uuid.UUID  `json:"id"`
	CreatedAt          time.Time  `json:"createdAt"`
	InstitutionID      uuid.UUID  `json:"institutionId"`
	UserID             uuid.UUID  `json:"userId"`
	ApplicationID      *uuid.UUID `json:"applicationId,omitempty"`
	Email              string     `json:"email"`
	PersonName         string     `json:"personName"`
	PersonFunction     string     `json:"personFunction"`
	InstitutionName    string     `json:"institutionName"`
	Operator           string     `json:"operator"`
	Street             string     `json:"street"`
	PostalCode         string     `json:"postalCode"`
	City               string     `json:"city"`
	DeclarationVersion string     `json:"declarationVersion"`
	DeclarationHash    string     `json:"declarationHash"`
}
