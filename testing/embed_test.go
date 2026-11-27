package testing

import (
	"fmt"
	"strings"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// EmbedTestSuite checks embedding a game on another website (#340): its
// embed link plays framed only on its organisation's verified page, after
// the host's yes; off stops the snippet and on revives it.
type EmbedTestSuite struct {
	testutil.BaseSuite
}

func TestEmbedSuite(t *testing.T) {
	s := &EmbedTestSuite{}
	s.SuiteName = "Embedding"
	suite.Run(t, s)
}

// world is an organisation whose head wrote a game in its workshop and
// gave a key to the organisation.
type world struct {
	org, game, share string
	head             *testutil.UserClient
}

func (s *EmbedTestSuite) world() world {
	org := Must(s.DevUser().CreateInstitution(testutil.Unique("Einbettung"))).Id.String()
	head := s.CreateUser(testutil.Unique("head"))
	invite := Must(s.DevUser().InviteToInstitution(org, "head", head.ID))
	Must(head.AcceptInviteByToken(invite.Id.String()))
	key := Must(head.AddApiKey(testutil.Unique("mock-embed"), "Embed key", "mock"))
	share := Must(head.ShareApiKeyWithInstitution(key.Id.String(), org)).Id.String()
	workshop := Must(head.CreateWorkshop(org, testutil.Unique("Werkstatt"))).Id.String()
	Must(head.SetActiveWorkshop(&workshop))
	game := Must(head.UploadGame("alien-first-contact")).Id.String()
	return world{org: org, game: game, share: share, head: head}
}

// A game embeds only for a verified page: there is no embedding on any
// website by the token alone.
func (s *EmbedTestSuite) TestAGameEmbedsOnlyForAVerifiedPage() {
	w := s.world()

	_, err := w.head.SetGameEmbed(w.game, testutil.Embedding{Enabled: true, Mode: "any", PaidBy: w.share})
	s.Invalid(err, "any website")
	_, err = w.head.SetGameEmbed(w.game, testutil.Embedding{Enabled: true, PaidBy: w.share})
	s.Invalid(err, "no verified page")
	s.Nil(Must(w.head.GetGameEmbed(w.game)))
}

// approvedPage is the organisation's verified page for host h, approved by
// the admin.
func (s *EmbedTestSuite) approvedPage(w world, h *testutil.Host) testutil.VerifiedPage {
	page := Must(w.head.CreateVerifiedPage(w.org, h.URL+"/play", h.URL+"/callback"))
	return Must(s.DevUser().ApproveVerifiedPage(page.Id.String(), true))
}

// A verified page starts nothing until an admin approves it, and a change
// of its site (scheme and host) withdraws the approval; port and path are free.
func (s *EmbedTestSuite) TestAVerifiedPagePlaysOnlyOnceApproved() {
	w := s.world()
	h := s.StartHost()
	page := Must(w.head.CreateVerifiedPage(w.org, h.URL+"/play", h.URL+"/callback"))
	s.Nil(page.ApprovedAt)
	embed := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{Enabled: true, PaidBy: w.share, VerifiedPage: page.Id.String()}))
	s.False(embed.VerifiedPageApproved)
	start := func(request string) error {
		_, err := s.Public().CreateSession(testutil.ForShare(embed.Token).FromHost("visitor-1", request))
		return err
	}

	s.Refused(start("a"), testutil.CodeHostRefused, "not approved yet")
	s.Contains(Must(s.Public().PlaybackPolicy("new&share="+embed.Token)), "frame-ancestors 'self'")

	_, err := w.head.ApproveVerifiedPage(page.Id.String(), true)
	s.Forbidden(err, "a head approves nothing")
	approved := Must(s.DevUser().ApproveVerifiedPage(page.Id.String(), true))
	s.NotNil(approved.ApprovedAt)
	s.True(Must(w.head.GetGameEmbed(w.game)).VerifiedPageApproved)
	MustSucceed(start("b"))

	// The admin approved the site: another path or port on it keeps the approval.
	moved := Must(w.head.UpdateVerifiedPage(page.Id.String(), h.OnAnotherPort()+"/elsewhere", h.URL+"/callback"))
	s.NotNil(moved.ApprovedAt, "a new path and port on the same site stay approved")
	s.Equal(page.PageId, moved.PageId, "the page ID the host checks stays across an edit")
	MustSucceed(start("c"))

	// Another host is another site, and waits for approval again.
	elsewhere := strings.Replace(h.URL, "127.0.0.1", "localhost", 1)
	changed := Must(w.head.UpdateVerifiedPage(page.Id.String(), elsewhere+"/play", elsewhere+"/callback"))
	s.Nil(changed.ApprovedAt, "a page on another site waits for approval again")
	s.Equal(page.PageId, changed.PageId, "the page ID stays on another site too")
	s.Refused(start("d"), testutil.CodeHostRefused, "changed")
}

// A page's callback answers for its site, so it is on that site; port and
// path are free.
func (s *EmbedTestSuite) TestAVerifiedPagesCallbackIsOnItsSite() {
	w := s.world()
	h := s.StartHost()
	elsewhere := strings.Replace(h.URL, "127.0.0.1", "localhost", 1)

	_, err := w.head.CreateVerifiedPage(w.org, h.URL+"/play", elsewhere+"/callback")
	s.Refused(err, testutil.CodeCallbackOffSite, "a callback on another host")
	page := Must(w.head.CreateVerifiedPage(w.org, h.OnAnotherPort()+"/play", h.URL+"/hooks/start"))
	s.Equal("http://127.0.0.1", page.Site, "the site has no port")
}

// A game its designer, head or staff, wrote outside any workshop is the
// organisation's too (#564): a colleague embeds it on the organisation's
// verified page.
func (s *EmbedTestSuite) TestAGameOfTheOrganisationThroughItsDesignerEmbeds() {
	w := s.world()
	Must(w.head.SetActiveWorkshop(nil))
	game := Must(w.head.UploadGame("alien-first-contact")).Id.String()
	staff := s.CreateUser(testutil.Unique("staff"))
	invite := Must(s.DevUser().InviteToInstitution(w.org, "staff", staff.ID))
	Must(staff.AcceptInviteByToken(invite.Id.String()))
	page := Must(w.head.CreateVerifiedPage(w.org, "https://example.org/play", "https://example.org/callback")).Id.String()

	read := Must(staff.ReadGame(game))
	s.Require().NotNil(read.OrganisationId)
	s.Equal(w.org, read.OrganisationId.String())
	embed := Must(staff.SetGameEmbed(game, testutil.Embedding{Enabled: true, PaidBy: w.share, VerifiedPage: page}))
	s.Require().NotNil(embed.VerifiedPageId)
	s.Equal(page, embed.VerifiedPageId.String())
}

// An organisation has one verified page; the admin sees every one.
func (s *EmbedTestSuite) TestAnOrganisationHasOneVerifiedPage() {
	w := s.world()
	page := Must(w.head.CreateVerifiedPage(w.org, "https://example.org/play", "https://example.org/callback"))
	_, err := w.head.CreateVerifiedPage(w.org, "https://example.org/other", "https://example.org/callback")
	s.Refused(err, testutil.CodeVerifiedPageExists)

	var listed []string
	for _, p := range Must(s.DevUser().ListAllVerifiedPages()) {
		listed = append(listed, p.Id.String())
	}
	s.Contains(listed, page.Id.String())
}

// Off stops the snippet; on revives the same one.
func (s *EmbedTestSuite) TestOffStopsTheSnippetAndOnRevivesIt() {
	w := s.world()
	h := s.StartHost()
	page := s.approvedPage(w, h).Id.String()
	first := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{Enabled: true, PaidBy: w.share, VerifiedPage: page}))
	s.Equal("verified", first.Mode)
	start := testutil.ForShare(first.Token).FromHost("visitor-1", "request-1")

	off := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{Enabled: false, VerifiedPage: page}))
	s.Equal(first.Token, off.Token)
	// refused as for a token that opens nothing
	_, err := s.Public().CreateSession(start)
	s.Refused(err, "denied")
	s.Contains(Must(s.Public().PlaybackPolicy("new&share="+first.Token)), "frame-ancestors 'self'")

	on := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{Enabled: true, VerifiedPage: page}))
	s.Equal(first.Token, on.Token)
	Must(s.Public().CreateSession(start))
}

// An ordinary share link does not embed: only the platform frames it.
func (s *EmbedTestSuite) TestAShareLinkDoesNotEmbed() {
	w := s.world()
	link := Must(w.head.CreateGameShare(w.game, w.share, nil))
	s.Contains(Must(s.Public().PlaybackPolicy("new&share="+link.Token)), "frame-ancestors 'self'")
}

// A stylesheet in the playback-engine's address loads, but what it loads in
// turn stays on the platform: anywhere, framed or not.
func (s *EmbedTestSuite) TestAStylesheetLoadsNothingFromAThirdSite() {
	w := s.world()
	link := Must(w.head.CreateGameShare(w.game, w.share, nil))
	policy := Must(s.Public().PlaybackPolicy("new&share=" + link.Token + "&css=https%3A%2F%2Fstyles.example%2Fgame.css"))
	s.Contains(policy, "img-src 'self' data: blob:;")
	s.Contains(policy, "font-src 'self' data: blob:;")
	s.Contains(policy, "style-src 'self' 'unsafe-inline' https://styles.example")
}

// A game embedded for a verified page plays only after its host's yes: a
// bare token, a no, a broken answer and silence start nothing.
func (s *EmbedTestSuite) TestAVerifiedPageStartsAGameOnlyOnItsHostsYes() {
	w := s.world()
	h := s.StartHost()
	page := s.approvedPage(w, h)
	s.Len(page.PageId, 16)
	s.Equal("http://127.0.0.1", page.Site)
	embed := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{
		Enabled: true, PaidBy: w.share, VerifiedPage: page.Id.String(),
	}))

	_, err := s.Public().CreateSession(testutil.ForShare(embed.Token))
	s.Refused(err, testutil.CodeHostRefused, "a bare token")

	Must(s.Public().CreateSession(testutil.ForShare(embed.Token).FromHost("visitor-1", "request-1")))
	s.Equal(map[string]string{"pageId": page.PageId, "playerId": "visitor-1", "requestId": "request-1"}, h.LastAsked())

	for name, answer := range map[string]func(){"a no": h.SaysNo, "a broken answer": h.Breaks, "silence": h.FallsSilent} {
		answer()
		_, err := s.Public().CreateSession(testutil.ForShare(embed.Token).FromHost("visitor-1", "request-2"))
		s.Refused(err, testutil.CodeHostRefused, name)
	}

	policy := Must(s.Public().PlaybackPolicy("new&share=" + embed.Token))
	// Any port on the approved site may frame the game.
	s.Contains(policy, "frame-ancestors http://127.0.0.1:*")
	s.Contains(policy, "img-src 'self' data: blob: http://127.0.0.1:*")
}

// embedToken is the token of a game embedded on an approved verified page.
func (s *EmbedTestSuite) embedToken() string {
	w := s.world()
	page := s.approvedPage(w, s.StartHost())
	return Must(w.head.SetGameEmbed(w.game, testutil.Embedding{
		Enabled: true, PaidBy: w.share, VerifiedPage: page.Id.String(),
	})).Token
}

// visit starts the embedded game for the host's visitor and answers the
// session.
func (s *EmbedTestSuite) visit(token, visitor, request string) string {
	return Must(s.Public().CreateSession(testutil.ForShare(token).FromHost(visitor, request)))
}

// guests is how many guests play the sessions.
func (s *EmbedTestSuite) guests(sessions ...string) int {
	return s.DatabaseCount(fmt.Sprintf("SELECT COUNT(DISTINCT user_id) FROM game_session WHERE id IN ('%s')", strings.Join(sessions, "','")))
}

// Every start on a verified page is a new game: a returning visitor is the
// same guest, playing a new session.
func (s *EmbedTestSuite) TestEveryStartOnAVerifiedPageIsANewGame() {
	token := s.embedToken()

	first := s.visit(token, "visitor-9", "a")
	again := s.visit(token, "visitor-9", "b")

	s.NotEqual(first, again, "a returning visitor starts a new game")
	s.Equal(1, s.guests(first, again), "a returning visitor is the same guest")
}

// Another visitor of a verified page is another guest.
func (s *EmbedTestSuite) TestAnotherVisitorIsAnotherGuest() {
	token := s.embedToken()

	first := s.visit(token, "visitor-9", "a")
	other := s.visit(token, "visitor-10", "b")

	s.Equal(2, s.guests(first, other), "another visitor is another guest")
}

// A visitor's guest exists only for embedded play: it holds no login code,
// email or login of its own to sign in with, and starting a game hands out
// nothing but the session's id.
func (s *EmbedTestSuite) TestAnEmbedGuestHasNoWayIntoThePlatform() {
	session := s.visit(s.embedToken(), "visitor-9", "a")

	guest := fmt.Sprintf("SELECT COUNT(*) FROM app_user u JOIN game_session g ON g.user_id = u.id WHERE g.id = '%s'", session)
	s.Require().Equal(1, s.DatabaseCount(guest+" AND u.outside_id IS NOT NULL"), "the visitor's guest")
	s.Equal(1, s.DatabaseCount(guest+" AND u.guest_token IS NULL AND u.email IS NULL AND u.auth0_id IS NULL"),
		"no login code, email or login")
}

// A game embeds only for a verified page of its own organisation.
func (s *EmbedTestSuite) TestAGameEmbedsOnlyForItsOrganisationsPages() {
	w, elsewhere := s.world(), s.world()
	page := Must(elsewhere.head.CreateVerifiedPage(elsewhere.org, "https://example.org/play", "https://example.org/callback"))

	_, err := w.head.SetGameEmbed(w.game, testutil.Embedding{
		Enabled: true, PaidBy: w.share, VerifiedPage: page.Id.String(),
	})
	s.Invalid(err)
}

// In development a verified page may be plain http on any host; the admin
// approves its site, scheme and host, as for https. Outside development both
// addresses are https, which the core's own tests pin down: a test instance
// always runs in development.
func (s *EmbedTestSuite) TestAVerifiedPageInDevelopmentMayBePlainHTTP() {
	w := s.world()
	page := Must(w.head.CreateVerifiedPage(w.org, "http://example.org/play", "http://example.org/callback"))
	s.Equal("http://example.org", page.Site)
	s.Nil(page.ApprovedAt, "plain http waits for the admin as https does")
}

// Each address that is no full web address is refused with a code naming it.
func (s *EmbedTestSuite) TestAVerifiedPagesAddressesAreFullAddresses() {
	w := s.world()
	_, err := w.head.CreateVerifiedPage(w.org, "example.org/play", "https://example.org/callback")
	s.Refused(err, testutil.CodePageURLInvalid, "a page address without scheme")
	_, err = w.head.CreateVerifiedPage(w.org, "https://example.org/play", "ftp://example.org/callback")
	s.Refused(err, testutil.CodeCallbackURLInvalid, "a callback on another scheme")
	_, err = w.head.CreateVerifiedPage(w.org, "https://example.org/play#start", "https://example.org/callback")
	s.Refused(err, testutil.CodePageURLInvalid, "a page address with a fragment")
}
