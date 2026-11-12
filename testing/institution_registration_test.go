package testing

import (
	"fmt"
	"strings"
	"testing"

	"cgl/api/routes"
	"cgl/legal"
	"cgl/obj"
	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// InstitutionRegistrationSuite covers organisation self-registration:
// new accounts, upgrades, e-mail confirmation, consent log and mails.
type InstitutionRegistrationSuite struct {
	testutil.BaseSuite
	counter int
}

func TestInstitutionRegistrationSuite(t *testing.T) {
	s := &InstitutionRegistrationSuite{}
	s.SuiteName = "Institution Registration Tests"
	suite.Run(t, s)
}

const testConstraint = "Erlaube leichte Spannung und einfache Konflikte; verhindere Gewalt oder Verletzungen aktiv."

// details returns a complete, unique organisation form.
func (s *InstitutionRegistrationSuite) details(email string) obj.InstitutionDetails {
	s.counter++
	return obj.InstitutionDetails{
		PersonName:       "Erika Muster",
		PersonFunction:   "Leitung Jugendzentrum",
		Email:            email,
		InstitutionName:  fmt.Sprintf("Jugendzentrum Test %d", s.counter),
		Operator:         "Stadt Teststadt",
		Street:           "Musterstraße 1",
		PostalCode:       "80331",
		City:             "München",
		PromptConstraint: testConstraint,
	}
}

func consents() routes.InstitutionConsents {
	return routes.InstitutionConsents{
		ConfirmAdult:       true,
		AcceptDeclaration:  true,
		DeclarationVersion: legal.CurrentDeclaration().Version,
	}
}

func (s *InstitutionRegistrationSuite) registerRequest(email string) routes.RegisterInstitutionRequest {
	s.counter++
	return routes.RegisterInstitutionRequest{
		InstitutionDetails:  s.details(email),
		InstitutionConsents: consents(),
		AccountName:         fmt.Sprintf("orga%d", s.counter),
		Language:            "de",
		AcceptTerms:         true,
		AcceptPrivacy:       true,
	}
}

func (s *InstitutionRegistrationSuite) confirm(token string) routes.ConfirmInstitutionResponse {
	var resp routes.ConfirmInstitutionResponse
	s.Public().MustPost("institution-applications/confirm", routes.ConfirmInstitutionRequest{Token: token}, &resp)
	return resp
}

// setAgeGroup sets the user's own age group (dev users start without one).
func setAgeGroup(u *testutil.UserClient, ageGroup string) {
	MustSucceed(u.Post("users/"+u.ID, map[string]any{"name": u.Name, "email": u.Email, "ageGroup": ageGroup}, nil))
}

// TestRegisterNewOrganisation: full path for a new person, from form to workshop.
func (s *InstitutionRegistrationSuite) TestRegisterNewOrganisation() {
	person := s.NewSignupClient()
	req := s.registerRequest("leitung-neu@jz-test.de")

	var resp routes.InstitutionApplicationResponse
	MustSucceed(person.Post("auth/register-institution", req, &resp))
	s.Equal(obj.InstitutionApplicationPending, resp.Application.Status)
	s.Equal("1.0", resp.Application.DeclarationVersion)

	// Account exists as adult individual; no organisation before the link is confirmed.
	me := Must(person.GetMe())
	s.Require().NotNil(me.AgeGroup)
	s.Equal(obj.AgeGroupU18, *me.AgeGroup)
	s.Equal(obj.RoleIndividual, me.Role.Role)
	s.Nil(me.Role.Institution)

	var mine routes.MyInstitutionApplicationResponse
	MustSucceed(person.Get("institution-applications/mine", &mine))
	s.Require().NotNil(mine.Application)
	s.Equal(req.InstitutionName, mine.Application.InstitutionName)

	token := s.LatestConfirmToken(req.Email)
	result := s.confirm(token)

	// Organisation exists, person is head, constraint and details are set.
	s.Equal(req.InstitutionName, result.Institution.Name)
	me = Must(person.GetMe())
	s.Equal(obj.RoleHead, me.Role.Role)
	s.Require().NotNil(me.Role.Institution)
	s.Equal(result.Institution.ID, me.Role.Institution.ID)

	inst := Must(person.GetInstitution(result.Institution.ID.String()))
	s.Require().NotNil(inst.PromptConstraints)
	s.Equal(testConstraint, *inst.PromptConstraints)

	// Consent log has the right version and hash.
	declaration := legal.CurrentDeclaration()
	s.Equal(declaration.Version, result.Consent.DeclarationVersion)
	s.Equal(declaration.Hash, result.Consent.DeclarationHash)
	s.Equal(req.Email, result.Consent.Email)
	s.Equal(req.PersonFunction, result.Consent.PersonFunction)
	s.Equal(req.Operator, result.Consent.Operator)

	// Confirmation mail to the person and a copy to the operator, both with the full text.
	var toPerson, toOperator bool
	for _, m := range s.DevMails() {
		if !strings.Contains(m.Subject, req.InstitutionName) {
			continue
		}
		s.Contains(m.Body, "10. Ich nehme zustimmend zur Kenntnis")
		s.Contains(m.Body, "Version:          1.0")
		switch strings.ToLower(m.To) {
		case req.Email:
			toPerson = true
		case "chatgamelab@jff.de":
			toOperator = true
		}
	}
	s.True(toPerson, "confirmation mail to the person")
	s.True(toOperator, "copy to chatgamelab@jff.de")

	// The link works only once.
	s.Public().FailPost("institution-applications/confirm", routes.ConfirmInstitutionRequest{Token: token},
		testutil.ErrorContains(obj.ErrCodeApplicationTokenInvalid))

	// Right away the head can create a workshop.
	ws := Must(person.CreateWorkshop(result.Institution.ID.String(), "Erster Workshop"))
	s.Equal("Erster Workshop", ws.Name)
}

// TestRegisterRequiresAllConsents: without 18+ or any of the three consents nothing is created.
func (s *InstitutionRegistrationSuite) TestRegisterRequiresAllConsents() {
	cases := map[string]func(r *routes.RegisterInstitutionRequest){
		"no 18+":           func(r *routes.RegisterInstitutionRequest) { r.ConfirmAdult = false },
		"no terms":         func(r *routes.RegisterInstitutionRequest) { r.AcceptTerms = false },
		"no privacy":       func(r *routes.RegisterInstitutionRequest) { r.AcceptPrivacy = false },
		"no declaration":   func(r *routes.RegisterInstitutionRequest) { r.AcceptDeclaration = false },
		"old version":      func(r *routes.RegisterInstitutionRequest) { r.DeclarationVersion = "0.9" },
		"empty constraint": func(r *routes.RegisterInstitutionRequest) { r.PromptConstraint = "" },
		"too long constraint": func(r *routes.RegisterInstitutionRequest) {
			r.PromptConstraint = strings.Repeat("a", obj.MaxPromptConstraintLength+1)
		},
		"no operator": func(r *routes.RegisterInstitutionRequest) { r.Operator = " " },
	}
	for name, mutate := range cases {
		s.Run(name, func() {
			person := s.NewSignupClient()
			req := s.registerRequest(fmt.Sprintf("consent-%d@jz-test.de", s.counter))
			mutate(&req)
			MustFail(person.Post("auth/register-institution", req, nil))
			// No account was created.
			_, err := person.GetMe()
			s.Require().Error(err)
			s.Contains(err.Error(), "user_not_registered")
		})
	}
}

// TestRegisterWithEmailOfExistingAccount: the address of an existing account cannot be used.
func (s *InstitutionRegistrationSuite) TestRegisterWithEmailOfExistingAccount() {
	existing := s.CreateUser("bestand")
	person := s.NewSignupClient()
	req := s.registerRequest(existing.Email)
	err := person.Post("auth/register-institution", req, nil)
	s.Require().Error(err)
	s.Contains(err.Error(), obj.ErrCodeEmailBelongsToOtherAccount)
	_, err = person.GetMe()
	s.Require().Error(err, "no account must be created")
}

// TestOrganisationNameMustBeFree: the same organisation name cannot be registered twice.
func (s *InstitutionRegistrationSuite) TestOrganisationNameMustBeFree() {
	first := s.NewSignupClient()
	req := s.registerRequest("name-erste@jz-test.de")
	MustSucceed(first.Post("auth/register-institution", req, nil))
	s.confirm(s.LatestConfirmToken(req.Email))

	second := s.NewSignupClient()
	req2 := s.registerRequest("name-zweite@jz-test.de")
	req2.InstitutionName = strings.ToUpper(req.InstitutionName)
	err := second.Post("auth/register-institution", req2, nil)
	s.Require().Error(err)
	s.Contains(err.Error(), obj.ErrCodeInstitutionNameTaken)
}

// TestUpgradeAdultAccount: an existing adult account becomes head; its games stay.
func (s *InstitutionRegistrationSuite) TestUpgradeAdultAccount() {
	user := s.CreateUser("upgrade18")
	setAgeGroup(user, obj.AgeGroupU18)
	game := Must(user.UploadGame("alien-first-contact"))

	req := routes.UpgradeInstitutionRequest{InstitutionDetails: s.details(user.Email), InstitutionConsents: consents()}
	var resp routes.InstitutionApplicationResponse
	MustSucceed(user.Post("institution-applications", req, &resp))
	s.Equal(obj.InstitutionApplicationUpgrade, resp.Application.Kind)
	s.Equal(obj.RoleIndividual, Must(user.GetMe()).Role.Role, "no organisation before confirmation")

	result := s.confirm(s.LatestConfirmToken(user.Email))
	me := Must(user.GetMe())
	s.Equal(obj.RoleHead, me.Role.Role)
	s.Equal(result.Institution.ID, me.Role.Institution.ID)

	still := Must(user.GetGameByID(game.ID.String()))
	s.Equal(game.ID, still.ID, "existing games stay")
}

// TestUpgradeRejectedForMinors: u13 and u13p accounts cannot upgrade.
func (s *InstitutionRegistrationSuite) TestUpgradeRejectedForMinors() {
	for _, age := range []string{obj.AgeGroupU13, obj.AgeGroupU13p} {
		user := s.CreateUser("minor" + age)
		setAgeGroup(user, age)
		req := routes.UpgradeInstitutionRequest{InstitutionDetails: s.details(user.Email), InstitutionConsents: consents()}
		err := user.Post("institution-applications", req, nil)
		s.Require().Error(err, age)
		s.Contains(err.Error(), obj.ErrCodeInstitutionAgeNotAllowed)
	}
}

// TestUpgradeRejectedForMembers: an account that already has an organisation role cannot upgrade.
func (s *InstitutionRegistrationSuite) TestUpgradeRejectedForMembers() {
	head := s.NewSignupClient()
	req := s.registerRequest("leitung-member@jz-test.de")
	MustSucceed(head.Post("auth/register-institution", req, nil))
	s.confirm(s.LatestConfirmToken(req.Email))

	upgrade := routes.UpgradeInstitutionRequest{InstitutionDetails: s.details("zweite-orga@jz-test.de"), InstitutionConsents: consents()}
	err := head.Post("institution-applications", upgrade, nil)
	s.Require().Error(err)
	s.Contains(err.Error(), obj.ErrCodeAlreadyInInstitution)
}

// TestUpgradeNeedsLogin: knowing an e-mail address is never enough.
func (s *InstitutionRegistrationSuite) TestUpgradeNeedsLogin() {
	user := s.CreateUser("ohnelogin")
	setAgeGroup(user, obj.AgeGroupU18)
	req := routes.UpgradeInstitutionRequest{InstitutionDetails: s.details(user.Email), InstitutionConsents: consents()}
	s.Public().FailPost("institution-applications", req)

	// Another logged-in account cannot apply with this account's address either.
	other := s.CreateUser("fremd")
	setAgeGroup(other, obj.AgeGroupU18)
	err := other.Post("institution-applications", req, nil)
	s.Require().Error(err)
	s.Contains(err.Error(), obj.ErrCodeEmailBelongsToOtherAccount)
}

// TestResendInvalidatesOldLink: a new link replaces the old one.
func (s *InstitutionRegistrationSuite) TestResendInvalidatesOldLink() {
	person := s.NewSignupClient()
	req := s.registerRequest("resend@jz-test.de")
	MustSucceed(person.Post("auth/register-institution", req, nil))
	oldToken := s.LatestConfirmToken(req.Email)

	MustSucceed(person.Post("institution-applications/resend", nil, nil))
	newToken := s.LatestConfirmToken(req.Email)
	s.NotEqual(oldToken, newToken)

	s.Public().FailPost("institution-applications/confirm", routes.ConfirmInstitutionRequest{Token: oldToken},
		testutil.ErrorContains(obj.ErrCodeApplicationTokenInvalid))
	s.confirm(newToken)
}

// TestUnknownToken: a made-up link does nothing.
func (s *InstitutionRegistrationSuite) TestUnknownToken() {
	s.Public().FailPost("institution-applications/confirm", routes.ConfirmInstitutionRequest{Token: "ausgedacht"},
		testutil.ErrorContains(obj.ErrCodeApplicationTokenInvalid))
}

// TestDeclarationEndpoint: the public endpoint serves the same declaration the consent log uses.
func (s *InstitutionRegistrationSuite) TestDeclarationEndpoint() {
	var resp routes.InstitutionDeclarationResponse
	s.Public().MustGet("legal/institution-declaration", &resp)
	s.Equal(legal.CurrentDeclaration().Hash, resp.Declaration.Hash)
	s.Equal("de", resp.BindingLanguage)
	s.Len(resp.YouthProtectionSuggestions, 7)
}
