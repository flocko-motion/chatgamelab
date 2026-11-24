package testing

import (
	"fmt"
	"strings"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// EmbedTestSuite checks embedding a game on another website (#340): its
// embed link plays framed on any website, or only on a verified page after
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

// A designer turns on embedding, and the game plays framed on any website
// by its token alone.
func (s *EmbedTestSuite) TestAGameEmbeddedForAnyWebsitePlaysThere() {
	w := s.world()

	embed := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{Enabled: true, PaidBy: w.share}))
	s.True(embed.Enabled)
	s.Equal("any", embed.Mode)
	s.NotEmpty(embed.Token)

	Must(s.Public().CreateSession(testutil.ForShare(embed.Token)))
	policy := Must(s.Public().PlaybackPolicy("new&share=" + embed.Token))
	s.Contains(policy, "frame-ancestors *")
}

// Off stops the snippet; on revives the same one.
func (s *EmbedTestSuite) TestOffStopsTheSnippetAndOnRevivesIt() {
	w := s.world()
	first := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{Enabled: true, PaidBy: w.share}))

	off := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{Enabled: false}))
	s.Equal(first.Token, off.Token)
	// refused as for a token that opens nothing
	_, err := s.Public().CreateSession(testutil.ForShare(first.Token))
	s.Refused(err, "denied")
	s.Contains(Must(s.Public().PlaybackPolicy("new&share="+first.Token)), "frame-ancestors 'self'")

	on := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{Enabled: true}))
	s.Equal(first.Token, on.Token)
	Must(s.Public().CreateSession(testutil.ForShare(first.Token)))
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
	page := Must(w.head.CreateVerifiedPage(w.org, h.URL+"/play", h.URL+"/callback"))
	s.Len(page.PageId, 16)
	s.Equal(h.URL, page.Origin)
	embed := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{
		Enabled: true, Verified: true, PaidBy: w.share, VerifiedPage: page.Id.String(),
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
	s.Contains(policy, "frame-ancestors "+h.URL)
	s.Contains(policy, "img-src 'self' data: blob: "+h.URL)
}

// A returning visitor of a verified page is the same guest; another
// visitor is another guest.
func (s *EmbedTestSuite) TestAReturningVisitorIsTheSameGuest() {
	w := s.world()
	h := s.StartHost()
	page := Must(w.head.CreateVerifiedPage(w.org, h.URL+"/play", h.URL+"/callback"))
	embed := Must(w.head.SetGameEmbed(w.game, testutil.Embedding{
		Enabled: true, Verified: true, PaidBy: w.share, VerifiedPage: page.Id.String(),
	}))

	first := Must(s.Public().CreateSession(testutil.ForShare(embed.Token).FromHost("visitor-9", "a")))
	again := Must(s.Public().CreateSession(testutil.ForShare(embed.Token).FromHost("visitor-9", "b")))
	other := Must(s.Public().CreateSession(testutil.ForShare(embed.Token).FromHost("visitor-10", "c")))

	guests := func(ids ...string) int {
		return s.DatabaseCount(fmt.Sprintf("SELECT COUNT(DISTINCT user_id) FROM game_session WHERE id IN ('%s')", strings.Join(ids, "','")))
	}
	s.Equal(1, guests(first, again), "the same visitor")
	s.Equal(2, guests(first, other), "another visitor")
}

// A game embeds only for a verified page of its own organisation.
func (s *EmbedTestSuite) TestAGameEmbedsOnlyForItsOrganisationsPages() {
	w, elsewhere := s.world(), s.world()
	page := Must(elsewhere.head.CreateVerifiedPage(elsewhere.org, "https://example.org/play", "https://example.org/callback"))

	_, err := w.head.SetGameEmbed(w.game, testutil.Embedding{
		Enabled: true, Verified: true, PaidBy: w.share, VerifiedPage: page.Id.String(),
	})
	s.Invalid(err)
}

// A verified page's address and callback are https.
func (s *EmbedTestSuite) TestAVerifiedPageIsHTTPS() {
	w := s.world()
	_, err := w.head.CreateVerifiedPage(w.org, "http://example.org/play", "https://example.org/callback")
	s.Invalid(err)
}
