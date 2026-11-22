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

// OrganisationView is an organisation as the caller sees it: the card
// always; settings, details and record only when the caller may see them,
// which hasSettings, hasDetails and hasRecord say. Field names follow v1's
// institution, so clients read the same keys.
type OrganisationView struct {
	ID   uuid.UUID        `json:"id"`
	Name string           `json:"name"`
	Meta OrganisationMeta `json:"meta"`

	// Members are its members (head and staff) as far as the
	// caller may list them; empty for one who may list none.
	Members []UserView `json:"members"`

	// The caller's rights on the organisation, by section, and on the whole
	// organisation. MayEditDetails is consenting to the declaration in its
	// name.
	MayEditCard     bool `json:"mayEditCard"`
	MayEditSettings bool `json:"mayEditSettings"`
	MayEditDetails  bool `json:"mayEditDetails"`
	MayDelete       bool `json:"mayDelete"`
	MaySuspend      bool `json:"maySuspend"`
	// Who may invite into it; who may, sees its invites too.
	MayInviteStaff bool `json:"mayInviteStaff"`
	MayInviteHead  bool `json:"mayInviteHead"`
	// Whether the caller may create its workshops, and list the keys given to it.
	MayCreateWorkshop bool `json:"mayCreateWorkshop"`
	MayListApiKeys    bool `json:"mayListApiKeys"`

	// HasSettings: the caller sees how the organisation runs.
	HasSettings          bool       `json:"hasSettings"`
	PromptConstraints    *string    `json:"promptConstraints,omitempty"`
	FreeUseApiKeyShareID *uuid.UUID `json:"freeUseApiKeyShareId,omitempty"`
	FreeUseAiQualityTier *string    `json:"freeUseAiQualityTier,omitempty"`

	// HasDetails: the caller sees who stands behind the organisation.
	HasDetails bool    `json:"hasDetails"`
	Operator   *string `json:"operator,omitempty"`
	Street     *string `json:"street,omitempty"`
	PostalCode *string `json:"postalCode,omitempty"`
	City       *string `json:"city,omitempty"`

	// HasRecord: the caller sees the organisation's standing.
	HasRecord       bool                          `json:"hasRecord"`
	Suspended       bool                          `json:"suspended"`
	SuspendedAt     *time.Time                    `json:"suspendedAt,omitempty"`
	SuspendedReason *string                       `json:"suspendedReason,omitempty"`
	Declaration     *ConsentStatusView            `json:"declaration,omitempty"`
	Registration    *OrganisationRegistrationView `json:"registration,omitempty"`
}

// OrganisationMeta is when an organisation was made and last changed.
type OrganisationMeta struct {
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// OrganisationRegistrationView is how an organisation registered itself:
// when, and the person who signed for it.
type OrganisationRegistrationView struct {
	At             time.Time `json:"at"`
	PersonName     string    `json:"personName"`
	PersonFunction string    `json:"personFunction"`
	Email          string    `json:"email"`
}

// OrganisationViewOf converts one organisation card, passing a loading error
// through:
//
//	return wire.OrganisationViewOf(core.ReadOrganisation(ctx, id))
func OrganisationViewOf(org *core.OrganisationCard, err error) (OrganisationView, error) {
	if err != nil {
		return OrganisationView{}, err
	}
	return organisationView(org), nil
}

// OrganisationViewsOf converts a list of organisation cards, passing a
// loading error through.
func OrganisationViewsOf(orgs []*core.OrganisationCard, err error) ([]OrganisationView, error) {
	if err != nil {
		return nil, err
	}
	views := make([]OrganisationView, len(orgs))
	for i, o := range orgs {
		views[i] = organisationView(o)
	}
	return views, nil
}

// organisationView converts an organisation card, with every section the
// caller may see.
func organisationView(org *core.OrganisationCard) OrganisationView {
	members, _ := UserViewsOf(org.Members(), nil)
	v := OrganisationView{
		ID:                org.ID(),
		Name:              org.Name(),
		Meta:              OrganisationMeta{CreatedAt: org.CreatedAt(), ModifiedAt: org.ModifiedAt()},
		Members:           members,
		MayEditCard:       org.MayEditCard(),
		MayEditSettings:   org.MayEditSettings(),
		MayEditDetails:    org.MayEditDetails(),
		MayDelete:         org.MayDelete(),
		MaySuspend:        org.MaySuspend(),
		MayInviteStaff:    org.MayInviteStaff(),
		MayInviteHead:     org.MayInviteHead(),
		MayCreateWorkshop: org.MayCreateWorkshop(),
		MayListApiKeys:    org.MayListKeys(),
	}
	if s, err := org.AsOrganisationSettings(); err == nil {
		v.HasSettings = true
		v.PromptConstraints = s.PromptConstraint()
		v.FreeUseApiKeyShareID = s.FreeUseKeyShare()
		v.FreeUseAiQualityTier = s.FreeUseTier()
	}
	if d, err := org.AsOrganisationDetails(); err == nil {
		v.HasDetails = true
		v.Operator, v.Street, v.PostalCode, v.City = d.Operator(), d.Street(), d.PostalCode(), d.City()
	}
	if r, err := org.AsOrganisationRecord(); err == nil {
		v.HasRecord = true
		v.Suspended = r.Suspended()
		v.SuspendedAt = r.SuspendedAt()
		v.SuspendedReason = r.SuspendedReason()
		consent := ConsentStatusView(r.Consent())
		v.Declaration = &consent
		if reg := r.Registration(); reg != nil {
			v.Registration = &OrganisationRegistrationView{
				At: reg.At, PersonName: reg.PersonName, PersonFunction: reg.PersonFunction, Email: reg.Email,
			}
		}
	}
	return v
}
