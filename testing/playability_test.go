package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// PlayabilityTestSuite checks that a game says whether the caller can start
// it, and why not, before anyone presses Play: on the game list, on the game
// page and on a share link's welcome screen.
type PlayabilityTestSuite struct {
	testutil.BaseSuite
}

func TestPlayabilitySuite(t *testing.T) {
	s := &PlayabilityTestSuite{}
	s.SuiteName = "Playability Tests"
	suite.Run(t, s)
}

// A player with a working key can play their game, as the list and the game
// page both say.
func (s *PlayabilityTestSuite) TestAGameWithAKeyIsPlayable() {
	user := s.CreateUser("playable-key")
	Must(user.AddApiKey("mock-playable-key", "Key", "mock"))
	gameID := Must(user.UploadGame("alien-first-contact")).Id.String()

	s.Equal(testutil.Playable, Must(user.ReadGame(gameID)).Playability)
	s.Equal(testutil.Playable, Must(user.GameInList(gameID)).Playability)
}

// Without a key, the game says so before Play, and the start agrees.
func (s *PlayabilityTestSuite) TestAGameWithoutAKeySaysSo() {
	user := s.CreateUser("playable-nokey")
	gameID := Must(user.UploadGame("alien-first-contact")).Id.String()

	s.Equal(testutil.CodeNoKey, Must(user.ReadGame(gameID)).Playability)
	s.Equal(testutil.CodeNoKey, Must(user.GameInList(gameID)).Playability)
	_, err := user.CreateSession(testutil.ForGame(gameID))
	s.Refused(err, testutil.CodeNoKey)
}

// A share link's welcome screen says whether a new session can start through
// it: playable while plays are left, no_plays once they are used up.
func (s *PlayabilityTestSuite) TestAShareLinkSaysWhenItsPlaysAreUsedUp() {
	user := s.CreateUser("playable-share")
	key := Must(user.AddApiKey("mock-playable-share", "Key", "mock"))
	gameID := Must(user.UploadGame("alien-first-contact")).Id.String()
	limit := 1
	share := Must(user.CreateGameShare(gameID, key.Id.String(), &limit))

	s.Equal(testutil.Playable, Must(s.Public().GuestGetGameInfo(share.Token)).Playability)

	Must(s.Public().CreateSession(testutil.ForShare(share.Token)))

	s.Equal(testutil.CodeNoPlays, Must(s.Public().GuestGetGameInfo(share.Token)).Playability)
}

// While the workshop a participant is in is paused, their games say so.
func (s *PlayabilityTestSuite) TestAPausedWorkshopSaysSo() {
	org := Must(s.DevUser().CreateInstitution("Pausenhof")).Id.String()
	head := s.CreateUser("playable-head")
	Must(head.AcceptInviteByToken(Must(s.DevUser().InviteToInstitution(org, "head", head.ID)).Id.String()))
	key := Must(head.AddApiKey("mock-playable-paused", "Key", "mock"))
	share := Must(head.ShareApiKeyWithInstitution(key.Id.String(), org)).Id.String()
	workshopID := Must(head.CreateWorkshop(org, "Pausenwerkstatt")).Id.String()
	Must(head.SetWorkshopApiKey(workshopID, &share))
	participant := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshopID)).InviteToken))
	gameID := Must(participant.UploadGame("alien-first-contact")).Id.String()

	s.Equal(testutil.Playable, Must(participant.ReadGame(gameID)).Playability)

	Must(head.EditWorkshop(workshopID, func(u *testutil.WorkshopUpdate) { u.IsPaused = true }))

	s.Equal(testutil.CodePaused, Must(participant.ReadGame(gameID)).Playability)
	s.Equal(testutil.CodePaused, Must(participant.GameInList(gameID)).Playability)
}
