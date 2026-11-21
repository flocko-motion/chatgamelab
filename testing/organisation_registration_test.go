package testing

import (
	"fmt"
	"strings"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// OrganisationRegistrationTestSuite tests organisations registering
// themselves with a new account, their heads consenting to the declaration
// and withdrawing it, and suspended organisations: by an admin, or for want
// of a consent.
type OrganisationRegistrationTestSuite struct {
	testutil.BaseSuite
	counter int
}

func TestOrganisationRegistrationSuite(t *testing.T) {
	s := &OrganisationRegistrationTestSuite{}
	s.SuiteName = "Organisation Registration Tests"
	suite.Run(t, s)
}

const youthProtection = "Erlaube leichte Spannung und einfache Konflikte; verhindere Gewalt oder Verletzungen aktiv."

// unique is a number no other call in the suite returns.
func (s *OrganisationRegistrationTestSuite) unique() int {
	s.counter++
	return s.counter
}

// signUp is a complete organisation sign-up under a name of its own,
// consenting to the current declaration.
func (s *OrganisationRegistrationTestSuite) signUp() testutil.OrganisationSignUp {
	declaration := Must(s.Public().GetDeclaration())
	return testutil.OrganisationSignUp{
		PersonName:         "Erika Muster",
		PersonFunction:     "Leitung Jugendzentrum",
		InstitutionName:    fmt.Sprintf("Jugendzentrum Test %d", s.unique()),
		Operator:           "Stadt Teststadt",
		Street:             "Musterstraße 1",
		PostalCode:         "80331",
		City:               "München",
		PromptConstraint:   youthProtection,
		AcceptDeclaration:  true,
		DeclarationVersion: declaration.Declaration.Version,
	}
}

// registration registers a new account as the head of signUp's
// organisation, reached at email.
func (s *OrganisationRegistrationTestSuite) registration(email string, signUp testutil.OrganisationSignUp) testutil.Registration {
	language, accepted := "de", true
	return testutil.Registration{
		Name:         fmt.Sprintf("orga%d", s.unique()),
		Email:        &email,
		AgeGroup:     "u18",
		Language:     &language,
		AcceptTerms:  &accepted,
		Organisation: &signUp,
	}
}

// registerOrganisation signs up a new person as the head of a new
// organisation and answers them with the organisation's id.
func (s *OrganisationRegistrationTestSuite) registerOrganisation(email string) (*testutil.UserClient, testutil.OrganisationSignUp, string) {
	newcomer := s.CreateRegisteringUser()
	signUp := s.signUp()
	user := Must(newcomer.Register(s.registration(email, signUp)))
	s.Require().NotNil(user.Role)
	s.Require().NotNil(user.Role.Institution)
	newcomer.ID = user.Id.String()
	return newcomer, signUp, user.Role.Institution.Id.String()
}

// adultIndividual is a new individual of age.
func (s *OrganisationRegistrationTestSuite) adultIndividual(name string) *testutil.UserClient {
	user := s.CreateUser(fmt.Sprintf("%s%d", name, s.unique()))
	MustSucceed(user.SetAgeGroup("u18"))
	return user
}

// mailsTo are the mails sent to the address whose subject names the
// organisation.
func (s *OrganisationRegistrationTestSuite) mailsTo(address, organisation string) []testutil.Mail {
	var mails []testutil.Mail
	for _, m := range s.MailsSent() {
		if strings.EqualFold(m.To, address) && strings.Contains(m.Subject, organisation) {
			mails = append(mails, m)
		}
	}
	return mails
}

// A new person registers as an organisation: the organisation exists at
// once, they are its head, its consent is recorded and confirmed by mail,
// with a copy to the operator, and they can run it right away.
func (s *OrganisationRegistrationTestSuite) TestRegisteringAsAnOrganisation() {
	head, signUp, institutionID := s.registerOrganisation("leitung-neu@jz-test.de")

	me := Must(head.WhoAmI())
	s.Equal("registered", me.State)
	s.Require().NotNil(me.User)
	s.Equal("head", me.User.Role.Role)
	s.Equal(signUp.InstitutionName, me.User.Role.Institution.Name)
	s.Require().NotNil(me.User.AgeGroup)
	s.Equal("u18", *me.User.AgeGroup)
	s.Empty(me.InstitutionSuspension, "the organisation consented when it registered")
	s.True(me.MayConsentForInstitution, "its head may consent for it")
	s.False(Must(head.ReadOrganisationConsent(institutionID)).Declaration.Pending)

	institution := Must(head.GetInstitution(institutionID))
	s.False(institution.Suspended)
	s.Require().NotNil(institution.PromptConstraints)
	s.Equal(youthProtection, *institution.PromptConstraints)
	s.Require().NotNil(institution.Operator)
	s.Equal(signUp.Operator, *institution.Operator)
	s.Require().NotNil(institution.City)
	s.Equal(signUp.City, *institution.City)

	toPerson := s.mailsTo("leitung-neu@jz-test.de", signUp.InstitutionName)
	s.Require().Len(toPerson, 1, "the consent is confirmed to the person")
	s.Contains(toPerson[0].Body, "10. Ich nehme zustimmend zur Kenntnis", "with the whole declaration")
	s.Contains(toPerson[0].Body, "Du bist Organisationsleitung", "in the person's language")
	s.Len(s.mailsTo("chatgamelab@jff.de", signUp.InstitutionName), 1, "a copy goes to the operator")

	workshop := Must(head.CreateWorkshop(institutionID, "Erster Workshop"))
	s.Equal("Erster Workshop", workshop.Name)
}

// A registration as an organisation is refused as a whole when anything is
// missing or wrong: the account stays registering.
func (s *OrganisationRegistrationTestSuite) TestRegisteringAsAnOrganisationIsValidated() {
	taken := s.signUp()
	Must(s.CreateRegisteringUser().Register(s.registration("name-erste@jz-test.de", taken)))

	cases := map[string]struct {
		change func(r *testutil.Registration)
		code   string
	}{
		"a minor":          {func(r *testutil.Registration) { r.AgeGroup = "u13p" }, testutil.CodeInstitutionAgeNotAllowed},
		"no terms":         {func(r *testutil.Registration) { r.AcceptTerms = nil }, testutil.CodeInstitutionConsentsRequired},
		"no email":         {func(r *testutil.Registration) { r.Email = nil }, testutil.CodeValidation},
		"no declaration":   {func(r *testutil.Registration) { r.Organisation.AcceptDeclaration = false }, testutil.CodeInstitutionConsentsRequired},
		"an old version":   {func(r *testutil.Registration) { r.Organisation.DeclarationVersion = "0.9" }, testutil.CodeDeclarationVersionOutdated},
		"no operator":      {func(r *testutil.Registration) { r.Organisation.Operator = " " }, testutil.CodeValidation},
		"no restriction":   {func(r *testutil.Registration) { r.Organisation.PromptConstraint = "" }, testutil.CodeValidation},
		"long restriction": {func(r *testutil.Registration) { r.Organisation.PromptConstraint = strings.Repeat("a", 201) }, testutil.CodeValidation},
		"a taken name": {func(r *testutil.Registration) {
			r.Organisation.InstitutionName = strings.ToUpper(taken.InstitutionName)
		}, testutil.CodeInstitutionNameTaken},
	}
	for name, c := range cases {
		s.Run(name, func() {
			newcomer := s.CreateRegisteringUser()
			r := s.registration(fmt.Sprintf("refused-%d@jz-test.de", s.unique()), s.signUp())
			c.change(&r)
			_, err := newcomer.Register(r)
			s.Refused(err, c.code)
			s.Equal("registering", Must(newcomer.WhoAmI()).State, "a refused registration leaves nothing behind")
		})
	}
}

// Another account's email is never accepted, ignoring case.
func (s *OrganisationRegistrationTestSuite) TestAnotherAccountsEmailIsRefused() {
	existing := s.CreateUser(fmt.Sprintf("bestand%d", s.unique()))
	newcomer := s.CreateRegisteringUser()
	_, err := newcomer.Register(s.registration(strings.ToUpper(existing.Email), s.signUp()))
	s.Refused(err, testutil.CodeEmailBelongsToOtherAccount)
	s.Equal("registering", Must(newcomer.WhoAmI()).State)
}

// consent is a head's consent to the current declaration for the
// institution, with its details.
func (s *OrganisationRegistrationTestSuite) consent(email string) testutil.DeclarationConsent {
	return testutil.DeclarationConsent{
		PersonName: "Hanna Leitung", PersonFunction: "Geschäftsführung", Email: email,
		Operator: "Träger e.V.", Street: "Devweg 1", PostalCode: "12345", City: "Devstadt",
		PromptConstraint: youthProtection, AcceptDeclaration: true,
		DeclarationVersion: Must(s.Public().GetDeclaration()).Declaration.Version,
	}
}

// suspension is why the caller's organisation is suspended, as they see it.
func (s *OrganisationRegistrationTestSuite) suspension(member *testutil.UserClient) string {
	me := Must(member.WhoAmI())
	if me.InstitutionSuspension == nil {
		return ""
	}
	return *me.InstitutionSuspension
}

// An organisation an admin created is suspended until one of its heads
// consents: its heads, staff and guests are refused, but a head may read and
// give the consent; that consent covers the organisation and stores its
// details.
func (s *OrganisationRegistrationTestSuite) TestAnExistingOrganisationIsSuspendedUntilAHeadConsents() {
	institution := Must(s.DevUser().CreateSuspendedInstitution(fmt.Sprintf("Altbestand %d", s.unique())))
	institutionID := institution.Id.String()
	head1 := s.memberOf(institutionID, "head")
	head2 := s.memberOf(institutionID, "head")
	staff := s.memberOf(institutionID, "staff")
	s.True(Must(s.DevUser().GetInstitution(institutionID)).Suspended)

	for name, member := range map[string]*testutil.UserClient{"head": head1, "staff": staff} {
		s.Equal(testutil.SuspendedNoConsent, s.suspension(member), name)
	}
	// a head still reads the organisation, to consent for it; staff see
	// nothing of it until the suspension ends
	Must(head1.GetInstitution(institutionID))
	_, err := staff.GetInstitution(institutionID)
	s.NotFound(err, "staff")
	_, err = staff.CreateGame("Gesperrt")
	s.Error(err, "a suspended member creates nothing")

	pending := Must(head1.ReadOrganisationConsent(institutionID))
	s.True(pending.Declaration.Pending)
	s.Nil(pending.Declaration.ConsentedVersion)
	s.Equal(institution.Name, pending.Name)

	refused := map[string]struct {
		change func(c *testutil.DeclarationConsent)
		code   string
	}{
		"not accepted":     {func(c *testutil.DeclarationConsent) { c.AcceptDeclaration = false }, testutil.CodeInstitutionConsentsRequired},
		"an old version":   {func(c *testutil.DeclarationConsent) { c.DeclarationVersion = "0.9" }, testutil.CodeDeclarationVersionOutdated},
		"no operator":      {func(c *testutil.DeclarationConsent) { c.Operator = "" }, testutil.CodeValidation},
		"no restriction":   {func(c *testutil.DeclarationConsent) { c.PromptConstraint = "  " }, testutil.CodeValidation},
		"long restriction": {func(c *testutil.DeclarationConsent) { c.PromptConstraint = strings.Repeat("a", 201) }, testutil.CodeValidation},
		"a bad email":      {func(c *testutil.DeclarationConsent) { c.Email = "hanna" }, testutil.CodeValidation},
	}
	for name, r := range refused {
		bad := s.consent("hanna@altbestand.test")
		r.change(&bad)
		_, err := head1.ConsentToDeclaration(institutionID, bad)
		s.Refused(err, r.code, name)
	}
	s.Equal(testutil.SuspendedNoConsent, s.suspension(head1), "refused consents change nothing")

	given := Must(head1.ConsentToDeclaration(institutionID, s.consent("hanna@altbestand.test")))
	s.False(given.Declaration.Pending)
	for name, member := range map[string]*testutil.UserClient{"head": head1, "other head": head2, "staff": staff} {
		s.Empty(s.suspension(member), "%s: one head's consent covers the organisation", name)
	}

	stored := Must(head1.GetInstitution(institutionID))
	s.False(stored.Suspended)
	s.Require().NotNil(stored.Operator)
	s.Equal("Träger e.V.", *stored.Operator)
	s.Require().NotNil(stored.PromptConstraints)
	s.Equal(youthProtection, *stored.PromptConstraints, "the restriction confirmed is the organisation's")

	confirmations := s.mailsTo("hanna@altbestand.test", institution.Name)
	s.Require().Len(confirmations, 1)
	s.Contains(confirmations[0].Subject, "confirmed (version "+given.Declaration.CurrentVersion+")", "worded for a consent to an existing organisation, in the head's language")
	s.NotContains(confirmations[0].Body, "**")
	s.Len(s.mailsTo("chatgamelab@jff.de", institution.Name), 1)
}

// A new version of the declaration suspends an organisation whose consent
// is to an older one, until a head consents to the new.
func (s *OrganisationRegistrationTestSuite) TestANewDeclarationSuspends() {
	institution := Must(s.DevUser().CreateSuspendedInstitution(fmt.Sprintf("Alte Fassung %d", s.unique())))
	institutionID := institution.Id.String()
	head := s.memberOf(institutionID, "head")
	s.ConsentedToAnOlderDeclaration(institutionID, institution.Name)

	s.Equal(testutil.SuspendedNoConsent, s.suspension(head))
	status := Must(head.ReadOrganisationConsent(institutionID)).Declaration
	s.True(status.Pending)
	s.Require().NotNil(status.ConsentedVersion)
	s.Equal("0.9", *status.ConsentedVersion)

	Must(head.ConsentToDeclaration(institutionID, s.consent(fmt.Sprintf("neu-%d@altbestand.test", s.unique()))))
	s.Empty(s.suspension(head))
}

// A head withdraws the organisation's consent, which suspends it until a head
// consents again; without a standing consent there is none to withdraw.
func (s *OrganisationRegistrationTestSuite) TestAHeadWithdrawsTheConsent() {
	head, _, institutionID := s.registerOrganisation(fmt.Sprintf("widerruf-%d@jz-test.de", s.unique()))
	staff := s.memberOf(institutionID, "staff")

	withdrawn := Must(head.WithdrawConsent(institutionID))
	s.True(withdrawn.Declaration.Pending)
	s.NotNil(withdrawn.Declaration.WithdrawnAt)
	s.Equal(testutil.SuspendedNoConsent, s.suspension(staff))
	_, err := staff.GetInstitution(institutionID)
	s.NotFound(err, "a suspended organisation's staff see nothing of it")

	_, err = head.WithdrawConsent(institutionID)
	s.Conflict(err, "nothing left to withdraw")

	Must(head.ConsentToDeclaration(institutionID, s.consent(fmt.Sprintf("wieder-%d@jz-test.de", s.unique()))))
	s.Empty(s.suspension(staff))
}

// Only a head of the organisation reads, gives or withdraws its consent.
func (s *OrganisationRegistrationTestSuite) TestOnlyItsHeadsConsentForAnOrganisation() {
	institution := Must(s.DevUser().CreateInstitution(fmt.Sprintf("Fremd %d", s.unique())))
	institutionID := institution.Id.String()
	staff := s.memberOf(institutionID, "staff")
	s.False(Must(staff.WhoAmI()).MayConsentForInstitution, "staff do not consent")

	otherHead, _, _ := s.registerOrganisation(fmt.Sprintf("andere-%d@jz-test.de", s.unique()))
	// those who see the organisation are refused
	for name, caller := range map[string]*testutil.UserClient{
		"staff":    staff,
		"an admin": s.DevUser(),
	} {
		_, err := caller.ReadOrganisationConsent(institutionID)
		s.Forbidden(err, name)
		_, err = caller.ConsentToDeclaration(institutionID, s.consent("x@fremd.test"))
		s.Forbidden(err, name)
		_, err = caller.WithdrawConsent(institutionID)
		s.Forbidden(err, name)
	}
	// for those outside it, it does not exist
	for name, caller := range map[string]*testutil.UserClient{
		"an individual":             s.adultIndividual("aussen"),
		"another organisation head": otherHead,
	} {
		_, err := caller.ReadOrganisationConsent(institutionID)
		s.NotFound(err, name)
		_, err = caller.ConsentToDeclaration(institutionID, s.consent("x@fremd.test"))
		s.NotFound(err, name)
		_, err = caller.WithdrawConsent(institutionID)
		s.NotFound(err, name)
	}
}

// memberOf is a new account the admin made a member of the institution in
// role.
func (s *OrganisationRegistrationTestSuite) memberOf(institutionID, role string) *testutil.UserClient {
	user := s.CreateUser(fmt.Sprintf("%s%d", role, s.unique()))
	invite := Must(s.DevUser().InviteToInstitution(institutionID, role, user.ID))
	Must(user.AcceptInviteByToken(invite.Id.String())) // a suspended organisation's member reads no invite after
	return user
}

// An admin's suspension stops the organisation's heads, staff and guests
// everywhere but /api/users/me, which reports it; it keeps their data, no
// consent lifts it, and only an admin does. Other organisations go on.
func (s *OrganisationRegistrationTestSuite) TestAnAdminSuspendsAnOrganisation() {
	admin := s.DevUser()
	head, _, institutionID := s.registerOrganisation(fmt.Sprintf("sperre-%d@jz-test.de", s.unique()))
	staff := s.memberOf(institutionID, "staff")
	workshop := Must(head.CreateWorkshop(institutionID, "Vor der Sperre"))
	guest := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshop.Id.String())).InviteToken))
	other, _, otherID := s.registerOrganisation(fmt.Sprintf("andere-%d@jz-test.de", s.unique()))

	_, err := head.SuspendInstitution(institutionID, "Test")
	s.Forbidden(err, "only an admin suspends")

	suspension := Must(admin.SuspendInstitution(institutionID, "Test"))
	s.NotNil(suspension.SuspendedAt)

	for name, member := range map[string]*testutil.UserClient{"head": head, "staff": staff, "guest": guest} {
		s.Equal(testutil.SuspendedByAdmin, s.suspension(member), name)
		_, err := member.GetWorkshop(workshop.Id.String())
		s.Refused(err, testutil.CodeOrganisationSuspended, name)
	}
	_, err = head.ConsentToDeclaration(institutionID, s.consent(fmt.Sprintf("trotzdem-%d@jz-test.de", s.unique())))
	MustSucceed(err)
	s.Equal(testutil.SuspendedByAdmin, s.suspension(head), "no consent lifts an admin's suspension")
	_, err = head.LiftSuspension(institutionID)
	s.Forbidden(err, "a head lifts no admin's suspension")

	s.Empty(s.suspension(other), "other organisations go on")
	Must(other.GetInstitution(otherID))

	var listed *testutil.Institution
	for _, institution := range Must(admin.ListInstitutions()) {
		if institution.Id.String() == institutionID {
			listed = &institution
		}
	}
	s.Require().NotNil(listed)
	s.True(listed.Suspended)
	s.NotNil(listed.SuspendedAt)
	s.Require().NotNil(listed.SuspendedReason)
	s.Equal("Test", *listed.SuspendedReason)
	s.Require().NotNil(listed.Declaration)
	s.False(listed.Declaration.Pending)
	s.Require().NotNil(listed.Operator)
	s.Equal("Träger e.V.", *listed.Operator, "as the head confirmed it")

	lifted := Must(admin.LiftSuspension(institutionID))
	s.Nil(lifted.SuspendedAt)
	s.Empty(s.suspension(head))
	s.Equal("Vor der Sperre", Must(head.GetWorkshop(workshop.Id.String())).Name, "nothing was lost")
}

// Admins see the organisations that registered themselves lately, with who
// signed for them; an organisation an admin created is not among them.
func (s *OrganisationRegistrationTestSuite) TestAdminsSeeNewRegistrations() {
	_, signUp, institutionID := s.registerOrganisation(fmt.Sprintf("antrag-%d@jz-test.de", s.unique()))
	byHand := Must(s.DevUser().CreateInstitution(fmt.Sprintf("Von Hand %d", s.unique())))
	s.True(Must(s.DevUser().WhoAmI()).MaySuspendInstitutions)

	recent := map[string]testutil.Institution{}
	for _, institution := range Must(s.DevUser().ListInstitutionsFiltered("recentlyRegistered")) {
		recent[institution.Id.String()] = institution
	}
	s.Require().Contains(recent, institutionID)
	s.NotContains(recent, byHand.Id.String(), "it did not register itself")
	registration := recent[institutionID].Registration
	s.Require().NotNil(registration)
	s.Equal(signUp.PersonName, registration.PersonName)
	s.Equal(signUp.PersonFunction, registration.PersonFunction)

	_, err := s.DevUser().ListInstitutionsFiltered("unknown")
	s.Invalid(err)
}

// An admin creating an organisation by hand gives its operator and address.
func (s *OrganisationRegistrationTestSuite) TestAnAdminCreatesAnOrganisationWithDetails() {
	operator, city := "Landkreis", "Kleinstadt"
	created := Must(s.DevUser().CreateInstitutionWith(testutil.InstitutionCreate{
		Name: fmt.Sprintf("Von Hand %d", s.unique()), Operator: &operator, City: &city,
	}))
	s.Require().NotNil(created.Operator)
	s.Equal(operator, *created.Operator)
	stored := Must(s.DevUser().GetInstitution(created.Id.String()))
	s.Require().NotNil(stored.City)
	s.Equal(city, *stored.City)
	s.Nil(stored.Street)
}

// The declaration is public: version, the binding German text with the
// sentence to tick, and the seven suggested restrictions.
func (s *OrganisationRegistrationTestSuite) TestTheDeclarationIsPublic() {
	declaration := Must(s.Public().GetDeclaration())
	s.Equal("de", declaration.BindingLanguage)
	s.True(strings.HasPrefix(declaration.Declaration.Text, "**Einverständniserklärung der Einrichtung"))
	s.NotEmpty(declaration.Declaration.Consent)
	s.Len(declaration.Declaration.Hash, 64)
	s.Len(declaration.YouthProtectionSuggestions, 7)
	s.Equal(200, declaration.MaxPromptConstraintLength)
}
