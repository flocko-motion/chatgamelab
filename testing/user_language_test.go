package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// UserLanguageTestSuite covers the language a newly created user ends up with.
//
// This is what a game session is locked to at launch, so getting it wrong means
// a German user sees a German interface but plays an English game.
type UserLanguageTestSuite struct {
	testutil.BaseSuite
}

func TestUserLanguageSuite(t *testing.T) {
	s := &UserLanguageTestSuite{}
	s.SuiteName = "User Language Tests"
	suite.Run(t, s)
}

// acceptInviteAnonymouslyWithLanguage joins a workshop through an invite link
// with the given Accept-Language header, as a browser does, and answers the
// new participant.
func (s *UserLanguageTestSuite) acceptInviteAnonymouslyWithLanguage(token, acceptLanguage string) testutil.AcceptedInvite {
	s.T().Helper()
	out := Must(s.AcceptWorkshopInviteAnonymouslyInLanguage(token, acceptLanguage))
	s.Require().NotNil(out.User)
	return out
}

// workshopInviteToken creates an institution, head and an active workshop, and
// returns an open participant invite token for it.
func (s *UserLanguageTestSuite) workshopInviteToken(prefix string) string {
	s.T().Helper()
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.Id.String()))

	// Anonymous joins require an active workshop with a usable API key.
	keyShare := Must(head.AddApiKey("mock-lang-key-"+prefix, prefix+" Key", "openai"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.Id.String(), inst.Id.String()))

	workshop := Must(head.CreateWorkshop(inst.Id.String(), prefix+" Workshop"))
	wsID := workshop.Id.String()
	orgShareID := orgShare.Id.String()
	Must(head.SetWorkshopApiKey(wsID, &orgShareID))
	Must(head.UpdateWorkshop(wsID, testutil.WorkshopUpdate{
		Name:                       prefix + " Workshop",
		Active:                     true,
		ShowOtherParticipantsGames: true,
	}))

	invite := Must(head.CreateWorkshopInvite(wsID))
	s.Require().NotNil(invite.InviteToken)
	return *invite.InviteToken
}

// A participant joining through a link never sees a registration form, so the
// browser's Accept-Language header is the only signal available.
func (s *UserLanguageTestSuite) TestParticipantInheritsBrowserLanguage() {
	token := s.workshopInviteToken("lang-de")

	resp := s.acceptInviteAnonymouslyWithLanguage(token, "de-DE,de;q=0.9,en;q=0.8")

	s.Equal("de", resp.User.Language,
		"participant with a German browser must be stored as German, otherwise their game starts in English")
}

// A region-only tag must still resolve to the base language.
func (s *UserLanguageTestSuite) TestParticipantLanguageStripsRegion() {
	token := s.workshopInviteToken("lang-region")

	resp := s.acceptInviteAnonymouslyWithLanguage(token, "de-AT")

	s.Equal("de", resp.User.Language)
}

// Unsupported or missing languages must fall back to English rather than
// storing something the game engine cannot use.
func (s *UserLanguageTestSuite) TestParticipantLanguageFallsBackToEnglish() {
	unsupported := s.workshopInviteToken("lang-unsupported")
	resp := s.acceptInviteAnonymouslyWithLanguage(unsupported, "zz-ZZ")
	s.Equal("en", resp.User.Language, "unsupported language must fall back to English")

	missing := s.workshopInviteToken("lang-missing")
	resp = s.acceptInviteAnonymouslyWithLanguage(missing, "")
	s.Equal("en", resp.User.Language, "missing header must fall back to English")
}

// The first supported entry wins, even when unsupported ones come first.
func (s *UserLanguageTestSuite) TestParticipantLanguagePicksFirstSupported() {
	token := s.workshopInviteToken("lang-order")

	resp := s.acceptInviteAnonymouslyWithLanguage(token, "zz,de;q=0.9,en;q=0.8")

	s.Equal("de", resp.User.Language)
}
