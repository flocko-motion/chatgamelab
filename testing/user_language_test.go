package testing

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"cgl/api/routes"
	"cgl/obj"
	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// UserLanguageTestSuite covers the language a newly created user ends up with.
//
// Workshop participants start in the workshop's language, which its leaders
// choose: shared workshop PCs often carry an English browser, so the browser's
// language is no reliable signal for a German workshop.
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
// with the given Accept-Language header, mimicking a browser.
//
// The shared PublicClient sends no custom headers, so this issues the request
// directly rather than widening the test harness for one case.
func (s *UserLanguageTestSuite) acceptInviteAnonymouslyWithLanguage(
	token string, acceptLanguage string,
) routes.AcceptInviteResponse {
	s.T().Helper()

	req, err := http.NewRequest(
		http.MethodPost,
		testutil.TestServerURL+"/api/invites/"+token+"/accept",
		bytes.NewReader(nil),
	)
	s.Require().NoError(err)
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Require().Equal(http.StatusOK, resp.StatusCode)

	var out routes.AcceptInviteResponse
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&out))
	s.Require().NotNil(out.User)
	return out
}

// workshopInviteToken creates an institution, a head speaking headLanguage and an
// active workshop, sets the workshop's language unless workshopLanguage is "",
// and returns the workshop and an open participant invite token for it.
func (s *UserLanguageTestSuite) workshopInviteToken(prefix, headLanguage, workshopLanguage string) (obj.Workshop, string) {
	s.T().Helper()
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	s.Require().NoError(head.SetUserLanguage(headLanguage))

	// Anonymous joins require an active workshop with a usable API key.
	keyShare := Must(head.AddApiKey("mock-lang-key-"+prefix, prefix+" Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))

	workshop := Must(head.CreateWorkshop(inst.ID.String(), prefix+" Workshop"))
	wsID := workshop.ID.String()
	orgShareID := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsID, &orgShareID))
	updates := map[string]interface{}{
		"name":                       prefix + " Workshop",
		"active":                     true,
		"public":                     false,
		"showPublicGames":            false,
		"showOtherParticipantsGames": true,
		"designEditingEnabled":       false,
		"isPaused":                   false,
	}
	if workshopLanguage != "" {
		updates["language"] = workshopLanguage
	}
	workshop = Must(head.UpdateWorkshop(wsID, updates))

	invite := Must(head.CreateWorkshopInvite(wsID, string(obj.RoleParticipant)))
	s.Require().NotNil(invite.InviteToken)
	return workshop, *invite.InviteToken
}

// A new workshop starts in its leader's language, and so do its participants,
// whatever their browser says.
func (s *UserLanguageTestSuite) TestWorkshopLanguageDefaultsToLeaderLanguage() {
	workshop, token := s.workshopInviteToken("lang-default", "de", "")
	s.Equal("de", workshop.Language, "a new workshop takes its creator's language")

	resp := s.acceptInviteAnonymouslyWithLanguage(token, "en-US,en;q=0.9")

	s.Equal("de", resp.User.Language,
		"a participant on an English browser must still start in the German workshop's language")
}

// The leaders may pick another language for new participants than their own.
func (s *UserLanguageTestSuite) TestParticipantStartsInChosenWorkshopLanguage() {
	workshop, token := s.workshopInviteToken("lang-chosen", "de", "fr")
	s.Equal("fr", workshop.Language)

	resp := s.acceptInviteAnonymouslyWithLanguage(token, "de-DE,de;q=0.9,en;q=0.8")

	s.Equal("fr", resp.User.Language)
}

// Without any browser language the participant still starts in the workshop's.
func (s *UserLanguageTestSuite) TestParticipantWithoutBrowserLanguage() {
	_, token := s.workshopInviteToken("lang-noheader", "en", "de")

	resp := s.acceptInviteAnonymouslyWithLanguage(token, "")

	s.Equal("de", resp.User.Language)
}

// Only languages the app supports can be chosen; codes are normalised.
func (s *UserLanguageTestSuite) TestWorkshopLanguageMustBeSupported() {
	workshop, _ := s.workshopInviteToken("lang-valid", "en", " DE ")
	s.Equal("de", workshop.Language)

	_, err := s.DevUser().UpdateWorkshop(workshop.ID.String(), map[string]interface{}{
		"name": "lang-valid Workshop", "active": true, "language": "zz",
	})
	s.Error(err, "an unsupported language must be refused")
}
