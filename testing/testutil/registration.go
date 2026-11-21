// package: testutil / the words for organisations registering themselves
// type:    logic
// job:     an organisation's consent to the declaration, given and withdrawn by its heads, admins suspending organisations, and the mails the platform sent
// limits:  institutions, workshops and invites otherwise live in orgs.go; registering is Register (users.go)
package testutil

import (
	"cgl/apiclient"

	"github.com/google/uuid"
)

// What a test sends and gets for organisations registering themselves.
type (
	OrganisationSignUp     = apiclient.OrganisationSignUpReq
	DeclarationConsent     = apiclient.ConsentToDeclarationReq
	Declaration            = apiclient.DeclarationResp
	OrganisationConsent    = apiclient.OrganisationConsentView
	OrganisationSuspension = apiclient.OrganisationSuspensionView
	Mail                   = apiclient.Message
)

// The reasons an organisation is suspended, as /api/users/me reports them.
const (
	SuspendedByAdmin   = "admin"
	SuspendedNoConsent = "consent"
)

// GetDeclaration is the declaration organisations consent to now.
func (c caller) GetDeclaration() (Declaration, error) {
	return decode[Declaration](c.api.GetApiLegalInstitutionDeclarationWithResponse(ctx()))
}

// ReadOrganisationConsent is the institution as its heads consent for it.
func (c caller) ReadOrganisationConsent(institutionID string) (OrganisationConsent, error) {
	return decode[OrganisationConsent](c.api.GetApiInstitutionsIdDeclarationConsentWithResponse(ctx(), institutionID))
}

// ConsentToDeclaration consents to the current declaration in the name of
// the institution.
func (c caller) ConsentToDeclaration(institutionID string, consent DeclarationConsent) (OrganisationConsent, error) {
	return decode[OrganisationConsent](c.api.PostApiInstitutionsIdDeclarationConsentWithResponse(ctx(), institutionID, consent))
}

// WithdrawConsent withdraws the institution's consent to the declaration.
func (c caller) WithdrawConsent(institutionID string) (OrganisationConsent, error) {
	return decode[OrganisationConsent](c.api.DeleteApiInstitutionsIdDeclarationConsentWithResponse(ctx(), institutionID))
}

// SuspendInstitution suspends an institution, noting why.
func (c caller) SuspendInstitution(institutionID, reason string) (OrganisationSuspension, error) {
	return decode[OrganisationSuspension](c.api.PostApiInstitutionsIdSuspensionWithResponse(ctx(), institutionID, apiclient.SuspendOrganisationReq{Reason: &reason}))
}

// LiftSuspension lifts an admin's suspension of an institution.
func (c caller) LiftSuspension(institutionID string) (OrganisationSuspension, error) {
	return decode[OrganisationSuspension](c.api.DeleteApiInstitutionsIdSuspensionWithResponse(ctx(), institutionID))
}

// ListInstitutionsFiltered lists the institutions the caller sees, narrowed
// by filters ("recentlyRegistered": those that registered themselves in the
// last two weeks).
func (c caller) ListInstitutionsFiltered(filters ...string) ([]Institution, error) {
	return decode[[]Institution](c.api.GetApiInstitutionsWithResponse(ctx(), &apiclient.GetApiInstitutionsParams{Filter: &filters}))
}

// SetAgeGroup sets the user's own age group.
func (u *UserClient) SetAgeGroup(ageGroup string) error {
	_, err := u.UpdateUser(u.ID, UserUpdate{AgeGroup: &ageGroup})
	return err
}

// MailsSent are the mails the platform sent since it started, newest first.
func (s *BaseSuite) MailsSent() []Mail {
	s.T().Helper()
	mails, err := decode[[]Mail](s.Public().api.GetApiDevMailsWithResponse(ctx()))
	if err != nil {
		s.T().Fatalf("reading the mails sent: %v", err)
	}
	return mails
}

// ConsentedToAnOlderDeclaration files a consent to an older version of the
// declaration for the institution, as if a new version came out after its
// head consented. The platform publishes no version on request, so the
// consent is written straight to the suite's database.
func (s *BaseSuite) ConsentedToAnOlderDeclaration(institutionID, name string) {
	s.T().Helper()
	s.consentOnFile(institutionID, name, "0.9", "an older text")
}

// consentOnFile writes a consent to the declaration in version and hash for
// the institution straight to the suite's database, given by a head no test
// knows: not the institution's creator, so it did not register itself.
func (s *BaseSuite) consentOnFile(institutionID, name, version, hash string) {
	s.T().Helper()
	_, err := s.inst.DB().Exec(`
		INSERT INTO institution_consent (
		  id, institution_id, user_id, kind, email, person_name, person_function,
		  institution_name, institution_operator, street, postal_code, city,
		  declaration_version, declaration_hash
		) VALUES ($1, $2, $3, 'consent', 'leitung@test.local', 'Test Leitung', 'Leitung',
		  $4, 'Träger', 'Weg 1', '12345', 'Teststadt', $5, $6)`,
		uuid.New(), institutionID, uuid.New(), name, version, hash)
	if err != nil {
		s.T().Fatalf("filing a consent for institution %s: %v", institutionID, err)
	}
}
