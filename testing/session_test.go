package testing

import (
	"strings"
	"testing"

	"cgl/testing/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// SessionTestSuite plays sessions through the engine mounted in the platform:
// creation, the one-time key, access, play counting, lazy restore and cleanup.
type SessionTestSuite struct {
	testutil.BaseSuite
}

func TestSessionSuite(t *testing.T) {
	s := &SessionTestSuite{}
	s.SuiteName = "Session Tests"
	suite.Run(t, s)
}

// player creates a user whose default key is the mock's, and a game of theirs.
// Returns (user, gameID, keyShareID).
func (s *SessionTestSuite) player(name string) (*testutil.UserClient, string, string) {
	user := s.CreateUser(name)
	key := Must(user.AddApiKey("mock-"+name, name+" Key", "mock"))
	game := Must(user.UploadGame("alien-first-contact"))
	return user, game.Id.String(), key.Id.String()
}

// --- Create and resume ---

func (s *SessionTestSuite) TestCreateAndResume() {
	user, gameID, _ := s.player("sess-create")

	id := Must(user.CreateSession(testutil.ForGame(gameID).InLanguage("de")))
	s.NotEmpty(id)

	Must(user.SessionTopology(id))

	sessions := Must(user.ListSessions())
	s.Require().Len(sessions, 1)
	s.Equal(id, sessions[0].Id.String())
	s.Equal("mock", sessions[0].Platform)
	s.NotEmpty(sessions[0].GameName)

	own := Must(user.GetGameSessions(gameID))
	s.Require().Len(own, 1)
	s.Equal(id, own[0].Id.String())
}

// Every created session is a play, the creator's own included, and shows in
// the player's profile.
func (s *SessionTestSuite) TestPlayCounting() {
	user, gameID, _ := s.player("sess-count")

	Must(user.CreateSession(testutil.ForGame(gameID)))

	game := Must(user.GetGameByID(gameID))
	s.Equal(1, game.PlayCount)
	stats := Must(user.GetStats())
	s.Equal(1, stats.GamesPlayed)
	s.Equal(1, stats.TotalPlaysOnGames)
}

// Starting a game anew replaces the player's earlier session of it.
func (s *SessionTestSuite) TestRestartReplacesTheEarlierSession() {
	user, gameID, _ := s.player("sess-restart")

	first := Must(user.CreateSession(testutil.ForGame(gameID)))
	second := Must(user.CreateSession(testutil.ForGame(gameID)))
	s.NotEqual(first, second)

	_, err := user.SessionTopology(first)
	s.Refused(err, testutil.CodeUnknown)
	s.Len(Must(user.ListSessions()), 1)
	s.Equal(2, Must(user.GetGameByID(gameID)).PlayCount)
}

// An input the player sends is counted as a message sent.
func (s *SessionTestSuite) TestMessagesSent() {
	user, gameID, _ := s.player("sess-input")
	id := Must(user.CreateSession(testutil.ForGame(gameID)))

	MustSucceed(user.Say(id, "I wave."))

	s.Equal(1, Must(user.GetStats()).MessagesSent)
}

// --- Launch errors ---

func (s *SessionTestSuite) TestNoKeyIsALaunchError() {
	user := s.CreateUser("sess-nokey")
	game := Must(user.UploadGame("alien-first-contact"))

	_, err := user.CreateSession(testutil.ForGame(game.Id.String()))
	s.Refused(err, testutil.CodeNoKey)
	s.Equal(0, Must(user.GetGameByID(game.Id.String())).PlayCount, "a refused launch is no play")
}

func (s *SessionTestSuite) TestShareWithNoPlaysLeft() {
	user, gameID, keyID := s.player("sess-noplays")
	limit := 1
	share := Must(user.CreateGameShare(gameID, keyID, &limit))

	Must(s.Public().CreateSession(testutil.ForShare(share.Token)))
	_, err := s.Public().CreateSession(testutil.ForShare(share.Token))
	s.Refused(err, testutil.CodeNoPlays)
}

// --- The one-time key ---

func (s *SessionTestSuite) TestOneTimeKeyAnswersTheSameSession() {
	user, gameID, _ := s.player("sess-once")
	key := uuid.NewString()

	first := Must(user.CreateSession(testutil.ForGame(gameID).Once(key)))
	again := Must(user.CreateSession(testutil.ForGame(gameID).Once(key)))

	s.Equal(first, again)
	s.Equal(1, Must(user.GetGameByID(gameID)).PlayCount)
}

// Another caller who knows the key creates nothing and learns no ID.
func (s *SessionTestSuite) TestOneTimeKeyBelongsToItsCaller() {
	user, gameID, _ := s.player("sess-once-owner")
	other := s.CreateUser("sess-once-other")
	key := uuid.NewString()

	first := Must(user.CreateSession(testutil.ForGame(gameID).Once(key)))

	_, err := other.CreateSession(testutil.ForGame(gameID).Once(key))
	s.Refused(err, testutil.CodeDenied)
	s.NotContains(err.Error(), first)
	s.Equal(1, Must(user.GetGameByID(gameID)).PlayCount)
	s.Empty(Must(other.ListSessions()))
}

// --- Access ---

// A session a signed-in account owns is reachable by that account only.
func (s *SessionTestSuite) TestAccountSessionIsTheOwners() {
	user, gameID, _ := s.player("sess-acc-owner")
	other := s.CreateUser("sess-acc-other")
	id := Must(user.CreateSession(testutil.ForGame(gameID)))

	Must(user.SessionState(id))

	_, err := other.SessionState(id)
	s.Refused(err, testutil.CodeDenied)
	_, err = s.Public().SessionState(id)
	s.Refused(err, testutil.CodeDenied)
	s.Refused(other.Say(id, "hijack"), testutil.CodeDenied)
}

// The platform refuses with schemaf's error body, exactly {"error", "code"},
// the code being the engine's problem.
func (s *SessionTestSuite) TestRefusalIsErrorAndCode() {
	user, gameID, _ := s.player("sess-refusal-body")
	other := s.CreateUser("sess-refusal-body-other")
	id := Must(user.CreateSession(testutil.ForGame(gameID)))

	_, err := other.SessionState(id)
	s.RefusedWithErrorAndCode(err, testutil.CodeDenied)
	_, err = s.Public().SessionState(id)
	s.RefusedWithErrorAndCode(err, testutil.CodeDenied)
}

// A guest's session is reachable with its id alone.
func (s *SessionTestSuite) TestGuestSessionIsReachableByID() {
	user, gameID, keyID := s.player("sess-guest")
	share := Must(user.CreateGameShare(gameID, keyID, nil))

	id := Must(s.Public().CreateSession(testutil.ForShare(share.Token).InLanguage("fr")))

	Must(s.Public().SessionState(id))
	Must(user.SessionState(id))
	s.Equal(1, Must(user.GetGameByID(gameID)).PlayCount)
}

// A share link played while signed in plays as that player.
func (s *SessionTestSuite) TestShareWhileSignedInPlaysAsThePlayer() {
	author, gameID, keyID := s.player("sess-share-author")
	share := Must(author.CreateGameShare(gameID, keyID, nil))
	player := s.CreateUser("sess-share-player")

	id := Must(player.CreateSession(testutil.ForShare(share.Token)))

	sessions := Must(player.ListSessions())
	s.Require().Len(sessions, 1)
	s.Equal(id, sessions[0].Id.String())

	_, err := s.Public().SessionState(id)
	s.Refused(err, testutil.CodeDenied)
	_, err = author.SessionState(id)
	s.Refused(err, testutil.CodeDenied)
}

// The playback-engine in its iframe sends a cookie, never a header: the
// session cookie names the player on the engine's routes.
func (s *SessionTestSuite) TestSessionCookie() {
	author, gameID, keyID := s.player("sess-cookie-author")
	share := Must(author.CreateGameShare(gameID, keyID, nil))

	inst := Must(s.DevUser().CreateInstitution("Cookie Org"))
	head := s.CreateUser("sess-cookie-head")
	Must(head.AcceptInvite(Must(s.DevUser().InviteToInstitution(inst.Id.String(), "head", head.ID)).Id.String()))
	workshop := Must(head.CreateWorkshop(inst.Id.String(), "Cookie Workshop"))
	invite := Must(head.CreateWorkshopInvite(workshop.Id.String()))

	browser := Must(s.JoinWorkshopWithCookie(*invite.InviteToken))
	id := Must(browser.CreateSession(testutil.ForShare(share.Token)))
	Must(browser.SessionState(id))

	// A cookie the platform never issued is a bad credential, not a guest.
	_, err := s.WithForgedCookie(browser).SessionState(id)
	s.Unauthorized(err)
}

// --- Lazy restore ---

// A session the server no longer holds in memory, as after a restart, is
// restored from storage by the first request that names it.
func (s *SessionTestSuite) TestLazyRestoreAfterRestart() {
	user, gameID, _ := s.player("sess-restore")
	id := Must(user.CreateSession(testutil.ForGame(gameID)))
	Must(user.SessionTopology(id))

	s.RestartBackend()

	Must(user.SessionTopology(id))
	Must(user.SessionHistory(id))
	MustSucceed(user.Say(id, "Still here?"))
}

// --- Cleanup ---

func (s *SessionTestSuite) TestDeletingAGameDeletesItsSessions() {
	user, gameID, _ := s.player("sess-del-game")
	id := Must(user.CreateSession(testutil.ForGame(gameID)))

	MustSucceed(user.DeleteGame(gameID))

	s.Empty(Must(user.ListSessions()))
	_, err := user.SessionState(id)
	s.Refused(err, testutil.CodeUnknown)
}

func (s *SessionTestSuite) TestDeletingAShareDeletesItsGuestsSessions() {
	user, gameID, keyID := s.player("sess-del-share")
	share := Must(user.CreateGameShare(gameID, keyID, nil))
	id := Must(s.Public().CreateSession(testutil.ForShare(share.Token)))

	MustSucceed(user.DeleteGameShare(gameID, share.Id.String()))

	_, err := s.Public().SessionState(id)
	s.Refused(err, testutil.CodeUnknown)
}

func (s *SessionTestSuite) TestDeletingAUserDeletesTheirSessions() {
	author, gameID, keyID := s.player("sess-del-author")
	share := Must(author.CreateGameShare(gameID, keyID, nil))
	player := s.CreateUser("sess-del-player")
	id := Must(player.CreateSession(testutil.ForShare(share.Token)))

	MustSucceed(s.DevUser().DeleteUser(player.ID))

	_, err := author.SessionState(id)
	s.Refused(err, testutil.CodeUnknown)
}

// A deleted key leaves the session, which resolves a key again on resume.
func (s *SessionTestSuite) TestDeletingAKeyKeepsTheSession() {
	user, gameID, keyID := s.player("sess-del-key")
	Must(user.CreateSession(testutil.ForGame(gameID)))

	MustSucceed(user.DeleteApiKey(keyID, true))

	sessions := Must(user.ListSessions())
	s.Require().Len(sessions, 1)
	s.Nil(sessions[0].ApiKeyShareId)
}

func (s *SessionTestSuite) TestDeletingASession() {
	user, gameID, _ := s.player("sess-del-own")
	other := s.CreateUser("sess-del-own-other")
	id := Must(user.CreateSession(testutil.ForGame(gameID)))

	s.NotFound(other.DeleteSession(id), "someone else's session does not exist for them")
	MustSucceed(user.DeleteSession(id))

	s.Empty(Must(user.ListSessions()))
	_, err := user.SessionState(id)
	s.Refused(err, testutil.CodeUnknown)
}

// --- Games' genre ---

func (s *SessionTestSuite) TestGameGenre() {
	user, gameID, _ := s.player("sess-genre")
	game := Must(user.GetGameByID(gameID))
	s.Equal("adventure", game.Genre)

	game.Genre = "npc-live"
	updated := Must(user.UpdateGame(gameID, game))
	s.Equal("npc-live", updated.Genre)

	game.Genre = "opera"
	_, err := user.UpdateGame(gameID, game)
	s.Invalid(err)

	yaml := Must(user.GetGameYAML(gameID))
	s.True(strings.Contains(yaml, "genre: npc-live"), yaml)

	s.NotEmpty(Must(user.ListGenres()))
	s.Equal("default", Must(user.ListThemes())[0].Id)
}
