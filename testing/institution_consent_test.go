package testing

import (
	"strings"

	"cgl/api/routes"
	"cgl/legal"
	"cgl/obj"
)

const (
	devHead1ID       = "00000000-0000-0000-0000-000000000003"
	devHead2ID       = "00000000-0000-0000-0000-000000000004"
	devStaff1ID      = "00000000-0000-0000-0000-000000000005"
	devIndividual1ID = "00000000-0000-0000-0000-000000000007"
	devInstitutionID = "00000000-0000-0000-0000-000000000010"
)

func consentRequest() routes.InstitutionConsentRequest {
	return routes.InstitutionConsentRequest{
		PersonName:         "Hanna Leitung",
		PersonFunction:     "Geschäftsführung",
		Email:              "leitung@dev-orga.test",
		Operator:           "Dev Träger e.V.",
		Street:             "Devweg 1",
		PostalCode:         "12345",
		City:               "Devstadt",
		AcceptDeclaration:  true,
		DeclarationVersion: legal.CurrentDeclaration().Version,
	}
}

// TestExistingOrganisationNeedsConsent: an organisation from before self-registration
// shows the pending declaration to its heads until one of them consents.
func (s *InstitutionRegistrationSuite) TestExistingOrganisationNeedsConsent() {
	head1 := s.DevUserClient(devHead1ID, "head-1")
	head2 := s.DevUserClient(devHead2ID, "head-2")

	me := Must(head1.GetMe())
	s.Require().NotNil(me.InstitutionDeclaration)
	s.True(me.InstitutionDeclaration.Pending)
	s.Nil(me.InstitutionDeclaration.ConsentedVersion)
	s.Equal(legal.CurrentDeclaration().Version, me.InstitutionDeclaration.CurrentVersion)

	// Incomplete or outdated consents are rejected.
	bad := consentRequest()
	bad.AcceptDeclaration = false
	MustFail(head1.Post("institutions/"+devInstitutionID+"/declaration-consent", bad, nil))
	bad = consentRequest()
	bad.DeclarationVersion = "0.9"
	MustFail(head1.Post("institutions/"+devInstitutionID+"/declaration-consent", bad, nil))
	bad = consentRequest()
	bad.Operator = ""
	MustFail(head1.Post("institutions/"+devInstitutionID+"/declaration-consent", bad, nil))
	s.True(Must(head1.GetMe()).InstitutionDeclaration.Pending, "rejected consents change nothing")

	var consent obj.InstitutionConsent
	MustSucceed(head1.Post("institutions/"+devInstitutionID+"/declaration-consent", consentRequest(), &consent))
	s.Equal(legal.CurrentDeclaration().Hash, consent.DeclarationHash)
	s.Nil(consent.ApplicationID)

	// One consent covers the whole organisation.
	for _, head := range []string{"head-1", "head-2"} {
		c := head1
		if head == "head-2" {
			c = head2
		}
		st := Must(c.GetMe()).InstitutionDeclaration
		s.Require().NotNil(st, head)
		s.False(st.Pending, head)
		s.Require().NotNil(st.ConsentedVersion)
		s.Equal(legal.CurrentDeclaration().Version, *st.ConsentedVersion)
	}

	// Operator and address are now stored on the organisation.
	inst := Must(head1.GetInstitution(devInstitutionID))
	s.Require().NotNil(inst.Operator)
	s.Equal("Dev Träger e.V.", *inst.Operator)
	s.Require().NotNil(inst.City)
	s.Equal("Devstadt", *inst.City)

	// Confirmation to the person and a copy, worded for a renewed consent.
	var toPerson, copy bool
	for _, m := range s.DevMails() {
		if !strings.Contains(m.Subject, "Einverständniserklärung für „Dev Organization“ bestätigt") {
			continue
		}
		s.Contains(m.Body, "10. Ich nehme zustimmend zur Kenntnis")
		s.NotContains(m.Body, "**")
		switch m.To {
		case "leitung@dev-orga.test":
			toPerson = true
		case "chatgamelab@jff.de":
			copy = true
		}
	}
	s.True(toPerson)
	s.True(copy)
}

// TestOnlyHeadsOfTheOrganisationCanConsent: staff, outsiders and heads of other organisations cannot.
func (s *InstitutionRegistrationSuite) TestOnlyHeadsOfTheOrganisationCanConsent() {
	staff := s.DevUserClient(devStaff1ID, "staff-1")
	err := staff.Post("institutions/"+devInstitutionID+"/declaration-consent", consentRequest(), nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "403")
	s.Nil(Must(staff.GetMe()).InstitutionDeclaration, "staff do not see the declaration status")

	individual := s.DevUserClient(devIndividual1ID, "individual-1")
	MustFail(individual.Post("institutions/"+devInstitutionID+"/declaration-consent", consentRequest(), nil))
	s.Nil(Must(individual.GetMe()).InstitutionDeclaration)

	// Head of a freshly registered organisation: nothing pending there, no access to the dev organisation.
	other := s.NewSignupClient()
	req := s.registerRequest("andere-leitung@jz-test.de")
	MustSucceed(other.Post("auth/register-institution", req, nil))
	s.confirm(s.LatestConfirmToken(req.Email))
	st := Must(other.GetMe()).InstitutionDeclaration
	s.Require().NotNil(st)
	s.False(st.Pending, "consent was given at registration")

	err = other.Post("institutions/"+devInstitutionID+"/declaration-consent", consentRequest(), nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "403")
}
