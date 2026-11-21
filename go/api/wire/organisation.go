package wire

import (
	"time"

	"cgl/core"

	"github.com/google/uuid"
)

// ConsentStatusView says whether an organisation's newest entry in the
// consent log covers the current declaration; pending (and the organisation
// suspended) while it has none, has withdrawn it, or consented to an older
// version.
type ConsentStatusView struct {
	CurrentVersion   string     `json:"currentVersion"`
	ConsentedVersion *string    `json:"consentedVersion,omitempty"`
	ConsentedAt      *time.Time `json:"consentedAt,omitempty"`
	WithdrawnAt      *time.Time `json:"withdrawnAt,omitempty"`
	Pending          bool       `json:"pending"`
}

// OrganisationConsentView is an organisation as its heads consent for it:
// its consent status, and the details and restriction a consent confirms.
type OrganisationConsentView struct {
	ID                uuid.UUID         `json:"id"`
	Name              string            `json:"name"`
	Operator          *string           `json:"operator,omitempty"`
	Street            *string           `json:"street,omitempty"`
	PostalCode        *string           `json:"postalCode,omitempty"`
	City              *string           `json:"city,omitempty"`
	PromptConstraints *string           `json:"promptConstraints,omitempty"`
	Declaration       ConsentStatusView `json:"declaration"`
}

// OrganisationConsentViewOf converts an organisation's consent, passing an
// error through.
func OrganisationConsentViewOf(c core.OrganisationConsent, err error) (OrganisationConsentView, error) {
	if err != nil {
		return OrganisationConsentView{}, err
	}
	return OrganisationConsentView{
		ID:                c.ID,
		Name:              c.Name,
		Operator:          c.Operator,
		Street:            c.Street,
		PostalCode:        c.PostalCode,
		City:              c.City,
		PromptConstraints: c.PromptConstraint,
		Declaration:       ConsentStatusView(c.Status),
	}, nil
}

// OrganisationSuspensionView is an admin's suspension of an organisation:
// when and why, both absent while there is none.
type OrganisationSuspensionView struct {
	ID              uuid.UUID  `json:"id"`
	SuspendedAt     *time.Time `json:"suspendedAt,omitempty"`
	SuspendedReason *string    `json:"suspendedReason,omitempty"`
}

// OrganisationSuspensionViewOf converts a suspension, passing an error
// through.
func OrganisationSuspensionViewOf(s core.OrganisationSuspension, err error) (OrganisationSuspensionView, error) {
	if err != nil {
		return OrganisationSuspensionView{}, err
	}
	return OrganisationSuspensionView{ID: s.ID, SuspendedAt: s.SuspendedAt, SuspendedReason: s.Reason}, nil
}
