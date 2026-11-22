package testing

import (
	"fmt"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// PlayAccessTestSuite checks who starts and goes on playing, as the secure
// core decides it (core/session.go).
type PlayAccessTestSuite struct {
	testutil.BaseSuite
	n int
}

func TestPlayAccessSuite(t *testing.T) {
	s := &PlayAccessTestSuite{}
	s.SuiteName = "Play access"
	suite.Run(t, s)
}

// workshop is a running workshop with a key, its head in it, and a game of
// the workshop.
func (s *PlayAccessTestSuite) workshop() (org, workshopID string, head *testutil.UserClient, gameID string) {
	s.n++
	org = Must(s.DevUser().CreateInstitution(fmt.Sprintf("Spiel %d", s.n))).Id.String()
	head = s.CreateUser(fmt.Sprintf("head%d", s.n))
	Must(head.AcceptInviteByToken(Must(s.DevUser().InviteToInstitution(org, "head", head.ID)).Id.String()))
	key := Must(head.AddApiKey(fmt.Sprintf("mock-play-%d", s.n), "Play key", "mock"))
	share := Must(head.ShareApiKeyWithInstitution(key.Id.String(), org)).Id.String()
	workshopID = Must(head.CreateWorkshop(org, fmt.Sprintf("Werkstatt %d", s.n))).Id.String()
	Must(head.SetWorkshopApiKey(workshopID, &share))
	Must(head.SetActiveWorkshop(&workshopID))
	gameID = Must(head.UploadGame("alien-first-contact")).Id.String()
	return org, workshopID, head, gameID
}

// pause pauses or resumes a workshop.
func (s *PlayAccessTestSuite) pause(head *testutil.UserClient, workshopID string, paused bool) {
	Must(head.EditWorkshop(workshopID, func(u *testutil.WorkshopUpdate) { u.IsPaused = paused }))
}

// While its workshop is paused, a session neither starts nor goes on; it
// does once the workshop resumes.
func (s *PlayAccessTestSuite) TestAPausedWorkshopHoldsItsGames() {
	_, workshopID, head, gameID := s.workshop()
	guest := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshopID)).InviteToken))
	id := Must(guest.CreateSession(testutil.ForGame(gameID)))

	s.pause(head, workshopID, true)
	_, err := guest.CreateSession(testutil.ForGame(gameID))
	s.Refused(err, testutil.CodePaused, "no new session")
	_, err = guest.SessionState(id)
	s.Refused(err, testutil.CodePaused, "no turn in a running one")

	s.pause(head, workshopID, false)
	Must(guest.SessionState(id))
}

// Head and staff play their organisation's workshop games outside workshop
// mode too: they run the workshops.
func (s *PlayAccessTestSuite) TestHeadsPlayTheirWorkshopsGames() {
	_, _, head, gameID := s.workshop()
	Must(head.SetActiveWorkshop(nil))
	Must(head.CreateSession(testutil.ForGame(gameID)))
}

// The admin plays any game, someone's private one included.
func (s *PlayAccessTestSuite) TestTheAdminPlaysAnyGame() {
	s.n++
	author := s.CreateUser(fmt.Sprintf("autor%d", s.n))
	game := Must(author.UploadGame("alien-first-contact")).Id.String()
	admin := s.DevUser()
	Must(admin.AddApiKey(fmt.Sprintf("mock-admin-%d", s.n), "Admin key", "mock"))
	Must(admin.CreateSession(testutil.ForGame(game)))
}

// A guest removed from their workshop goes on playing nothing there.
func (s *PlayAccessTestSuite) TestARemovedGuestStops() {
	_, workshopID, head, gameID := s.workshop()
	guest := Must(s.JoinWorkshop(*Must(head.CreateWorkshopInvite(workshopID)).InviteToken))
	id := Must(guest.CreateSession(testutil.ForGame(gameID)))

	s.NoError(head.RemoveMemberFromWorkshop(workshopID, guest.ID))
	_, err := guest.SessionState(id)
	s.Forbidden(err, "a removed guest plays no more")
}
