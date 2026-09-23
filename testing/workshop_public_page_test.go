package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// WorkshopPublicPageTestSuite tests the public workshop page /w/<slug>: its
// generated link, editing it, what visitors see, downloads, and the share links
// visitors play through.
type WorkshopPublicPageTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopPublicPageSuite(t *testing.T) {
	s := &WorkshopPublicPageTestSuite{}
	s.SuiteName = "Workshop Public Page Tests"
	suite.Run(t, s)
}

type publicPageFixture struct {
	head        *testutil.UserClient
	participant *testutil.UserClient
	wsID        string
	wsName      string
	slug        string
}

// setup creates an institution with a head, a workshop with a working key, and
// one participant. withKey false leaves the workshop without a key.
func (s *WorkshopPublicPageTestSuite) setup(prefix string, withKey bool) publicPageFixture {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	name := prefix + " Workshop"
	workshop := Must(head.CreateWorkshop(inst.ID.String(), name))
	wsID := workshop.ID.String()
	if withKey {
		keyShare := Must(head.AddApiKey("mock-"+prefix, prefix+" Key", "mock"))
		orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))
		orgShareID := orgShare.ID.String()
		Must(head.SetWorkshopApiKey(wsID, &orgShareID))
	}

	invite := Must(head.CreateWorkshopInvite(wsID, string(obj.RoleParticipant)))
	resp := Must(s.AcceptWorkshopInviteAnonymously(*invite.InviteToken))
	participant := s.CreateUserWithToken(*resp.AuthToken)

	ws := Must(head.GetWorkshop(wsID))
	s.Require().NotNil(ws.PublicSlug, "a new workshop must have a public page link")
	return publicPageFixture{head: head, participant: participant, wsID: wsID, wsName: name, slug: *ws.PublicSlug}
}

func (s *WorkshopPublicPageTestSuite) updateWorkshop(f publicPageFixture, public bool, extra map[string]interface{}) (obj.Workshop, error) {
	updates := map[string]interface{}{"name": f.wsName, "active": true, "public": public}
	for k, v := range extra {
		updates[k] = v
	}
	return f.head.UpdateWorkshop(f.wsID, updates)
}

// setGamePublic sends the whole game back, because the update replaces every field.
func (s *WorkshopPublicPageTestSuite) setGamePublic(u *testutil.UserClient, gameID string, public bool) {
	game := Must(u.GetGameByID(gameID))
	game.Public = public
	Must(u.UpdateGame(gameID, game))
}

func (s *WorkshopPublicPageTestSuite) page(slug string) (obj.PublicWorkshopPage, error) {
	var page obj.PublicWorkshopPage
	err := s.Public().Get("public/workshops/"+slug, &page)
	return page, err
}

func (s *WorkshopPublicPageTestSuite) getRaw(path string) (int, string) {
	resp, err := http.Get(fmt.Sprintf("%s/api/%s", testutil.TestServerURL, path))
	s.Require().NoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	return resp.StatusCode, string(body)
}

func (s *WorkshopPublicPageTestSuite) TestSlugFromWorkshopName() {
	f := s.setup("Robotik-AG: Ärger & Spaß", false)
	s.Regexp(regexp.MustCompile(`^robotik-ag-aerger-spass-[a-z]{4,10}-[a-z]{4,10}$`), f.slug)
}

func (s *WorkshopPublicPageTestSuite) TestPageOnlyWhilePublic() {
	f := s.setup("pubpage-toggle", true)
	_, err := s.page(f.slug)
	s.Error(err, "a workshop that is not public has no page")
	s.Contains(err.Error(), "404")

	Must(s.updateWorkshop(f, true, map[string]interface{}{"publicDescription": "  Wir bauen Spiele.\nZwei Tage lang.  "}))
	page := Must(s.page(f.slug))
	s.Equal(f.wsName, page.Name)
	s.Equal("Wir bauen Spiele.\nZwei Tage lang.", page.Description)
	s.Empty(page.Games)

	// Typed with capitals still finds the page.
	Must(s.page(strings.ToUpper(f.slug)))

	_, err = s.page("gibt-es-nicht-hier")
	s.Contains(err.Error(), "404")
}

func (s *WorkshopPublicPageTestSuite) TestListsOnlyPublicGamesWithoutCreators() {
	f := s.setup("pubpage-games", true)
	Must(s.updateWorkshop(f, true, nil))

	shown := Must(f.participant.UploadGame("alien-first-contact"))
	hidden := Must(f.participant.UploadGame("alien-first-contact"))
	s.setGamePublic(f.participant, shown.ID.String(), true)

	// A public game from another workshop stays off this page.
	other := s.setup("pubpage-games-other", true)
	foreign := Must(other.participant.UploadGame("alien-first-contact"))
	s.setGamePublic(other.participant, foreign.ID.String(), true)

	page := Must(s.page(f.slug))
	s.Require().Len(page.Games, 1)
	s.Equal(shown.ID, page.Games[0].ID)
	s.Equal(shown.Name, page.Games[0].Name)
	s.Require().NotNil(page.Games[0].Play)
	s.Equal(50, page.Games[0].Play.Remaining)
	s.Equal(50, page.Games[0].Play.Limit)

	status, body := s.getRaw("public/workshops/" + f.slug)
	s.Equal(http.StatusOK, status)
	var raw map[string]interface{}
	s.Require().NoError(json.Unmarshal([]byte(body), &raw))
	s.NotContains(body, "creator", "the page must never name who made a game")
	s.NotContains(body, "institution")

	// Download: listed games only.
	status, body = s.getRaw("public/workshops/" + f.slug + "/games/" + shown.ID.String() + "/yaml")
	s.Equal(http.StatusOK, status)
	s.Contains(body, shown.Name)
	status, _ = s.getRaw("public/workshops/" + f.slug + "/games/" + hidden.ID.String() + "/yaml")
	s.Equal(http.StatusNotFound, status)
	status, _ = s.getRaw("public/workshops/" + f.slug + "/games/" + foreign.ID.String() + "/yaml")
	s.Equal(http.StatusNotFound, status)
}

func (s *WorkshopPublicPageTestSuite) TestNoPlayWithoutWorkshopKey() {
	f := s.setup("pubpage-nokey", false)
	Must(s.updateWorkshop(f, true, nil))
	game := Must(f.participant.UploadGame("alien-first-contact"))
	s.setGamePublic(f.participant, game.ID.String(), true)

	page := Must(s.page(f.slug))
	s.Require().Len(page.Games, 1)
	s.Nil(page.Games[0].Play)
}

func (s *WorkshopPublicPageTestSuite) TestPlayCountsDownAndStaysApartFromHandMadeLinks() {
	f := s.setup("pubpage-play", true)
	Must(s.updateWorkshop(f, true, nil))
	game := Must(f.participant.UploadGame("alien-first-contact"))
	s.setGamePublic(f.participant, game.ID.String(), true)

	play := Must(s.page(f.slug)).Games[0].Play
	s.Require().NotNil(play)
	Must(s.Public().GuestCreateSession(play.Token))

	again := Must(s.page(f.slug)).Games[0].Play
	s.Equal(play.Token, again.Token, "the page keeps one link per game")
	s.Equal(49, again.Remaining)

	// A leader's workshop share is a separate, unlimited link.
	handMade := Must(f.head.CreateWorkshopGameShare(game.ID.String(), f.wsID, nil))
	s.NotEqual(play.Token, handMade.Token)
	s.Nil(handMade.Remaining)
}

func (s *WorkshopPublicPageTestSuite) TestLinksRevoked() {
	f := s.setup("pubpage-revoke", true)
	Must(s.updateWorkshop(f, true, nil))
	game := Must(f.participant.UploadGame("alien-first-contact"))
	s.setGamePublic(f.participant, game.ID.String(), true)

	// Unpublishing the game revokes its link.
	token := Must(s.page(f.slug)).Games[0].Play.Token
	s.setGamePublic(f.head, game.ID.String(), false)
	_, err := s.Public().GuestGetGameInfo(token)
	s.Error(err, "an unpublished game's page link must stop working")

	// Switching the page off revokes every link; switching it on makes fresh ones.
	s.setGamePublic(f.head, game.ID.String(), true)
	token = Must(s.page(f.slug)).Games[0].Play.Token
	Must(s.updateWorkshop(f, false, nil))
	_, err = s.Public().GuestGetGameInfo(token)
	s.Error(err, "switching the page off must revoke its links")

	Must(s.updateWorkshop(f, true, nil))
	fresh := Must(s.page(f.slug)).Games[0].Play
	s.NotEqual(token, fresh.Token)
	s.Equal(50, fresh.Remaining)

	// Deleting the workshop revokes them as well.
	MustSucceed(f.head.DeleteWorkshop(f.wsID))
	_, err = s.Public().GuestGetGameInfo(fresh.Token)
	s.Error(err, "deleting the workshop must revoke its page links")
	_, err = s.page(f.slug)
	s.Contains(err.Error(), "404")
}

func (s *WorkshopPublicPageTestSuite) TestEditSlug() {
	f := s.setup("pubpage-slug", true)
	Must(s.updateWorkshop(f, true, nil))

	ws := Must(s.updateWorkshop(f, true, map[string]interface{}{"publicSlug": "  Unser-Spiele-Workshop "}))
	s.Equal("unser-spiele-workshop", *ws.PublicSlug)
	Must(s.page("unser-spiele-workshop"))
	_, err := s.page(f.slug)
	s.Contains(err.Error(), "404", "the old link must stop working")

	// Other settings leave the link alone when publicSlug is omitted.
	ws = Must(s.updateWorkshop(f, true, map[string]interface{}{"isPaused": true}))
	s.Equal("unser-spiele-workshop", *ws.PublicSlug)

	for _, bad := range []string{"ab", "mit leerzeichen", "-vorne", "hinten-", "doppel--strich", "ümlaut", strings.Repeat("a", 61)} {
		_, err := s.updateWorkshop(f, true, map[string]interface{}{"publicSlug": bad})
		s.Error(err, bad)
		s.Contains(err.Error(), "public_slug_invalid", bad)
	}

	other := s.setup("pubpage-slug-other", true)
	_, err = s.updateWorkshop(other, false, map[string]interface{}{"publicSlug": "unser-spiele-workshop"})
	s.Error(err)
	s.Contains(err.Error(), "public_slug_taken")

	_, err = s.updateWorkshop(f, true, map[string]interface{}{"publicDescription": strings.Repeat("ä", 2001)})
	s.Error(err)
	s.Contains(err.Error(), "public_description_too_long")
	Must(s.updateWorkshop(f, true, map[string]interface{}{"publicDescription": strings.Repeat("ä", 2000)}))
}

func (s *WorkshopPublicPageTestSuite) TestParticipantCannotChangePage() {
	f := s.setup("pubpage-perm", true)
	_, err := f.participant.UpdateWorkshop(f.wsID, map[string]interface{}{
		"name": f.wsName, "active": true, "public": true,
	})
	s.Error(err)
	s.Contains(err.Error(), "403")
}
