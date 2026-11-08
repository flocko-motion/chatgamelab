package testing

import (
	"net/http"
	"strings"
	"testing"

	"cgl/testing/testutil"
	"engine"
	engineapi "engine/api"

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
	return user, game.ID.String(), key.ID.String()
}

func once() string { return uuid.NewString() }

// --- Create and resume ---

func (s *SessionTestSuite) TestCreateAndResume() {
	user, gameID, _ := s.player("sess-create")

	id, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Language: "de", Once: once()})
	s.Require().NoError(err)
	s.NotEmpty(id)

	topology := Must(user.Engine(http.MethodGet, "sessions/"+id+"/topology", nil))
	s.Equal(http.StatusOK, topology.Status, string(topology.Body))

	sessions := Must(user.ListSessions())
	s.Require().Len(sessions, 1)
	s.Equal(id, sessions[0].ID.String())
	s.Equal("mock", sessions[0].Platform)
	s.NotEmpty(sessions[0].GameName)

	own := Must(user.GetGameSessions(gameID))
	s.Require().Len(own, 1)
	s.Equal(id, own[0].ID.String())
}

// Every created session is a play, the creator's own included, and shows in
// the player's profile.
func (s *SessionTestSuite) TestPlayCounting() {
	user, gameID, _ := s.player("sess-count")

	_, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: once()})
	s.Require().NoError(err)

	game := Must(user.GetGameByID(gameID))
	s.Equal(1, game.PlayCount)
	stats := Must(user.GetStats())
	s.Equal(1, stats.GamesPlayed)
	s.Equal(1, stats.TotalPlaysOnGames)
}

// Starting a game anew replaces the player's earlier session of it.
func (s *SessionTestSuite) TestRestartReplacesTheEarlierSession() {
	user, gameID, _ := s.player("sess-restart")

	first, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: once()})
	s.Require().NoError(err)
	second, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: once()})
	s.Require().NoError(err)
	s.NotEqual(first, second)

	gone := Must(user.Engine(http.MethodGet, "sessions/"+first+"/topology", nil))
	s.Equal(http.StatusNotFound, gone.Status)
	s.Equal(engine.ProblemUnknown, gone.Refusal().Problem)
	s.Len(Must(user.ListSessions()), 1)
	s.Equal(2, Must(user.GetGameByID(gameID)).PlayCount)
}

// An input the player sends is counted as a message sent.
func (s *SessionTestSuite) TestMessagesSent() {
	user, gameID, _ := s.player("sess-input")
	id, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: once()})
	s.Require().NoError(err)

	resp := Must(user.Engine(http.MethodPost, "sessions/"+id+"/input", map[string]string{"text": "I wave."}))
	s.Require().Equal(http.StatusAccepted, resp.Status, string(resp.Body))

	s.Equal(1, Must(user.GetStats()).MessagesSent)
}

// --- Launch errors ---

func (s *SessionTestSuite) TestNoKeyIsALaunchError() {
	user := s.CreateUser("sess-nokey")
	game := Must(user.UploadGame("alien-first-contact"))

	_, resp, err := user.CreateSession(engineapi.CreateRequest{Game: game.ID.String(), Once: once()})
	s.Require().Error(err)
	s.Equal(engine.ProblemNoKey, resp.Refusal().Problem, string(resp.Body))
	s.Equal(0, Must(user.GetGameByID(game.ID.String())).PlayCount, "a refused launch is no play")
}

func (s *SessionTestSuite) TestShareWithNoPlaysLeft() {
	user, gameID, keyID := s.player("sess-noplays")
	limit := 1
	share := Must(user.CreateGameShare(gameID, keyID, &limit))

	_, _, err := s.Public().CreateSession(engineapi.CreateRequest{Share: share.Token, Once: once()})
	s.Require().NoError(err)
	_, resp, err := s.Public().CreateSession(engineapi.CreateRequest{Share: share.Token, Once: once()})
	s.Require().Error(err)
	s.Equal(engine.ProblemNoPlays, resp.Refusal().Problem, string(resp.Body))
}

// --- The one-time key ---

func (s *SessionTestSuite) TestOneTimeKeyAnswersTheSameSession() {
	user, gameID, _ := s.player("sess-once")
	key := once()

	first, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: key})
	s.Require().NoError(err)
	again, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: key})
	s.Require().NoError(err)

	s.Equal(first, again)
	s.Equal(1, Must(user.GetGameByID(gameID)).PlayCount)
}

// Another caller who knows the key creates nothing and learns no ID.
func (s *SessionTestSuite) TestOneTimeKeyBelongsToItsCaller() {
	user, gameID, _ := s.player("sess-once-owner")
	other := s.CreateUser("sess-once-other")
	key := once()

	first, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: key})
	s.Require().NoError(err)

	_, resp, err := other.CreateSession(engineapi.CreateRequest{Game: gameID, Once: key})
	s.Require().Error(err)
	s.Equal(http.StatusForbidden, resp.Status)
	s.Equal(engine.ProblemDenied, resp.Refusal().Problem)
	s.NotContains(string(resp.Body), first)
	s.Equal(1, Must(user.GetGameByID(gameID)).PlayCount)
	s.Empty(Must(other.ListSessions()))
}

// --- Access ---

// A session a signed-in account owns is reachable by that account only.
func (s *SessionTestSuite) TestAccountSessionIsTheOwners() {
	user, gameID, _ := s.player("sess-acc-owner")
	other := s.CreateUser("sess-acc-other")
	id, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: once()})
	s.Require().NoError(err)

	s.Equal(http.StatusOK, Must(user.Engine(http.MethodGet, "sessions/"+id+"/state", nil)).Status)

	denied := Must(other.Engine(http.MethodGet, "sessions/"+id+"/state", nil))
	s.Equal(http.StatusForbidden, denied.Status)
	s.Equal(engine.ProblemDenied, denied.Refusal().Problem)

	anonymous := Must(s.Public().Engine(http.MethodGet, "sessions/"+id+"/state", nil))
	s.Equal(http.StatusForbidden, anonymous.Status)

	input := Must(other.Engine(http.MethodPost, "sessions/"+id+"/input", map[string]string{"text": "hijack"}))
	s.Equal(http.StatusForbidden, input.Status)
}

// A guest's session is reachable with its id alone.
func (s *SessionTestSuite) TestGuestSessionIsReachableByID() {
	user, gameID, keyID := s.player("sess-guest")
	share := Must(user.CreateGameShare(gameID, keyID, nil))

	id, _, err := s.Public().CreateSession(engineapi.CreateRequest{Share: share.Token, Language: "fr", Once: once()})
	s.Require().NoError(err)

	s.Equal(http.StatusOK, Must(s.Public().Engine(http.MethodGet, "sessions/"+id+"/state", nil)).Status)
	s.Equal(http.StatusOK, Must(user.Engine(http.MethodGet, "sessions/"+id+"/state", nil)).Status)
	s.Equal(1, Must(user.GetGameByID(gameID)).PlayCount)
}

// A share link played while signed in plays as that player.
func (s *SessionTestSuite) TestShareWhileSignedInPlaysAsThePlayer() {
	author, gameID, keyID := s.player("sess-share-author")
	share := Must(author.CreateGameShare(gameID, keyID, nil))
	player := s.CreateUser("sess-share-player")

	id, _, err := player.CreateSession(engineapi.CreateRequest{Share: share.Token, Once: once()})
	s.Require().NoError(err)

	sessions := Must(player.ListSessions())
	s.Require().Len(sessions, 1)
	s.Equal(id, sessions[0].ID.String())

	s.Equal(http.StatusForbidden, Must(s.Public().Engine(http.MethodGet, "sessions/"+id+"/state", nil)).Status)
	s.Equal(http.StatusForbidden, Must(author.Engine(http.MethodGet, "sessions/"+id+"/state", nil)).Status)
}

// The playback-engine in its iframe sends a cookie, never a header: the play
// cookie names the player on the engine's routes.
func (s *SessionTestSuite) TestPlayCookie() {
	user, gameID, _ := s.player("sess-cookie")
	cookie := Must(user.PlayCookie())
	s.Equal("/api/engine", cookie.Path)
	s.True(cookie.HttpOnly)
	s.Equal(http.SameSiteStrictMode, cookie.SameSite)

	resp := Must(testutil.EngineWithCookie(cookie, http.MethodPost, "sessions",
		engineapi.CreateRequest{Game: gameID, Once: once()}))
	s.Require().Equal(http.StatusCreated, resp.Status, string(resp.Body))
	id := Must(user.ListSessions())[0].ID.String()

	s.Equal(http.StatusOK, Must(testutil.EngineWithCookie(cookie, http.MethodGet, "sessions/"+id+"/state", nil)).Status)

	forged := &http.Cookie{Name: "cgl_play", Value: "forged"}
	s.Equal(http.StatusForbidden, Must(testutil.EngineWithCookie(forged, http.MethodGet, "sessions/"+id+"/state", nil)).Status)
}

// --- Lazy restore ---

// A session the server no longer holds in memory, as after a restart, is
// restored from storage by the first request that names it.
func (s *SessionTestSuite) TestLazyRestoreAfterRestart() {
	user, gameID, _ := s.player("sess-restore")
	id, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: once()})
	s.Require().NoError(err)
	before := Must(user.Engine(http.MethodGet, "sessions/"+id+"/topology", nil))
	s.Require().Equal(http.StatusOK, before.Status)

	s.RestartBackend()

	after := Must(user.Engine(http.MethodGet, "sessions/"+id+"/topology", nil))
	s.Equal(http.StatusOK, after.Status, string(after.Body))
	history := Must(user.Engine(http.MethodGet, "sessions/"+id+"/history", nil))
	s.Equal(http.StatusOK, history.Status, string(history.Body))
	input := Must(user.Engine(http.MethodPost, "sessions/"+id+"/input", map[string]string{"text": "Still here?"}))
	s.Equal(http.StatusAccepted, input.Status, string(input.Body))
}

// --- Cleanup ---

func (s *SessionTestSuite) TestDeletingAGameDeletesItsSessions() {
	user, gameID, _ := s.player("sess-del-game")
	id, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: once()})
	s.Require().NoError(err)

	MustSucceed(user.DeleteGame(gameID))

	s.Empty(Must(user.ListSessions()))
	s.Equal(http.StatusNotFound, Must(user.Engine(http.MethodGet, "sessions/"+id+"/state", nil)).Status)
}

func (s *SessionTestSuite) TestDeletingAShareDeletesItsGuestsSessions() {
	user, gameID, keyID := s.player("sess-del-share")
	share := Must(user.CreateGameShare(gameID, keyID, nil))
	id, _, err := s.Public().CreateSession(engineapi.CreateRequest{Share: share.Token, Once: once()})
	s.Require().NoError(err)

	MustSucceed(user.DeleteGameShare(gameID, share.ID.String()))

	s.Equal(http.StatusNotFound, Must(s.Public().Engine(http.MethodGet, "sessions/"+id+"/state", nil)).Status)
}

func (s *SessionTestSuite) TestDeletingAUserDeletesTheirSessions() {
	author, gameID, keyID := s.player("sess-del-author")
	share := Must(author.CreateGameShare(gameID, keyID, nil))
	player := s.CreateUser("sess-del-player")
	id, _, err := player.CreateSession(engineapi.CreateRequest{Share: share.Token, Once: once()})
	s.Require().NoError(err)

	MustSucceed(s.DevUser().DeleteUser(player.ID))

	s.Equal(http.StatusNotFound, Must(author.Engine(http.MethodGet, "sessions/"+id+"/state", nil)).Status)
}

// A deleted key leaves the session, which resolves a key again on resume.
func (s *SessionTestSuite) TestDeletingAKeyKeepsTheSession() {
	user, gameID, keyID := s.player("sess-del-key")
	_, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: once()})
	s.Require().NoError(err)

	MustSucceed(user.Delete("apikeys/" + keyID))

	sessions := Must(user.ListSessions())
	s.Require().Len(sessions, 1)
	s.Nil(sessions[0].ApiKeyShareID)
}

func (s *SessionTestSuite) TestDeletingASession() {
	user, gameID, _ := s.player("sess-del-own")
	other := s.CreateUser("sess-del-own-other")
	id, _, err := user.CreateSession(engineapi.CreateRequest{Game: gameID, Once: once()})
	s.Require().NoError(err)

	MustFail(other.Delete("sessions/" + id))
	MustSucceed(user.Delete("sessions/" + id))

	s.Empty(Must(user.ListSessions()))
	s.Equal(http.StatusNotFound, Must(user.Engine(http.MethodGet, "sessions/"+id+"/state", nil)).Status)
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
	s.Error(err)

	yaml := Must(user.GetGameYAML(gameID))
	s.True(strings.Contains(yaml, "genre: npc-live"), yaml)

	var genres []engine.GenreInfo
	s.Require().NoError(user.Get("genres", &genres))
	s.NotEmpty(genres)
	var themes []engine.ThemeInfo
	s.Require().NoError(user.Get("themes", &themes))
	s.Equal("default", themes[0].ID)
}
