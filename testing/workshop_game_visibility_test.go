package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// WorkshopGameVisibilityTestSuite tests that workshop settings
// (showOtherParticipantsGames, showPublicGames) correctly control
// what games participants can see.
type WorkshopGameVisibilityTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopGameVisibilitySuite(t *testing.T) {
	s := &WorkshopGameVisibilityTestSuite{}
	s.SuiteName = "Workshop Game Visibility Tests"
	suite.Run(t, s)
}

// setupWorkshop creates an institution, head, workshop, and two participants.
// Returns (head, participant1, participant2, workshopID).
func (s *WorkshopGameVisibilityTestSuite) setupWorkshop(prefix string) (*testutil.UserClient, *testutil.UserClient, *testutil.UserClient, string) {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	workshop := Must(head.CreateWorkshop(inst.ID.String(), prefix+" Workshop"))
	wsIDStr := workshop.ID.String()

	invite1 := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp1, err := s.AcceptWorkshopInviteAnonymously(*invite1.InviteToken)
	s.NoError(err)
	p1 := s.CreateUserWithToken(*resp1.AuthToken)

	invite2 := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	resp2, err := s.AcceptWorkshopInviteAnonymously(*invite2.InviteToken)
	s.NoError(err)
	p2 := s.CreateUserWithToken(*resp2.AuthToken)

	s.T().Logf("Setup: head=%s, p1=%s, p2=%s, ws=%s", head.Name, p1.Name, p2.Name, wsIDStr)
	return head, p1, p2, wsIDStr
}

// TestOtherParticipantGamesHiddenByDefault verifies that by default
// (showOtherParticipantsGames=false), a participant cannot see another
// participant's game in the list.
func (s *WorkshopGameVisibilityTestSuite) TestOtherParticipantGamesHiddenByDefault() {
	_, p1, p2, _ := s.setupWorkshop("vis-hide")

	// P1 creates a game
	game := Must(p1.UploadGame("alien-first-contact"))
	s.T().Logf("P1 created game: %s", game.ID)

	// P1 can see their own game
	p1Games := Must(p1.ListGames())
	s.True(containsGame(p1Games, game.ID), "p1 should see their own game")

	// P2 should NOT see P1's game (default: showOtherParticipantsGames=false)
	p2Games := Must(p2.ListGames())
	s.False(containsGame(p2Games, game.ID), "p2 should NOT see p1's game when showOtherParticipantsGames is false")
	s.T().Logf("P2 sees %d games — correctly hidden", len(p2Games))
}

// TestOtherParticipantGamesVisibleWhenEnabled verifies that when
// showOtherParticipantsGames=true, a participant CAN see another
// participant's game.
func (s *WorkshopGameVisibilityTestSuite) TestOtherParticipantGamesVisibleWhenEnabled() {
	head, p1, p2, wsIDStr := s.setupWorkshop("vis-show")

	// Enable showOtherParticipantsGames
	ws := Must(head.GetWorkshop(wsIDStr))
	Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":                       ws.Name,
		"active":                     true,
		"showOtherParticipantsGames": true,
	}))
	s.T().Logf("Enabled showOtherParticipantsGames")

	// P1 creates a game
	game := Must(p1.UploadGame("alien-first-contact"))
	s.T().Logf("P1 created game: %s", game.ID)

	// P2 should now see P1's game
	p2Games := Must(p2.ListGames())
	s.True(containsGame(p2Games, game.ID), "p2 should see p1's game when showOtherParticipantsGames is true")
	s.T().Logf("P2 sees %d games — correctly visible", len(p2Games))
}

// TestPublicGamesHiddenByDefault verifies that by default
// (showPublicGames=false), a participant cannot see public games.
func (s *WorkshopGameVisibilityTestSuite) TestPublicGamesHiddenByDefault() {
	_, _, p2, _ := s.setupWorkshop("vis-pub-hide")

	// Create a public game from an outside user
	outsider := s.CreateUser("vis-pub-outsider")
	game := Must(outsider.UploadGame("alien-first-contact"))
	// Make it public
	Must(outsider.UpdateGame(game.ID.String(), map[string]interface{}{
		"name":   game.Name,
		"public": true,
	}))
	s.T().Logf("Outsider created public game: %s", game.ID)

	// P2 should NOT see the public game (default: showPublicGames=false)
	p2Games := Must(p2.ListGames())
	s.False(containsGame(p2Games, game.ID), "participant should NOT see public games when showPublicGames is false")
	s.T().Logf("P2 sees %d games — public game correctly hidden", len(p2Games))
}

// TestPublicGamesVisibleWhenEnabled verifies that when
// showPublicGames=true, a participant CAN see public games.
func (s *WorkshopGameVisibilityTestSuite) TestPublicGamesVisibleWhenEnabled() {
	head, _, p2, wsIDStr := s.setupWorkshop("vis-pub-show")

	// Enable showPublicGames
	ws := Must(head.GetWorkshop(wsIDStr))
	Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":            ws.Name,
		"active":          true,
		"showPublicGames": true,
	}))
	s.T().Logf("Enabled showPublicGames")

	// Create a public game from an outside user
	outsider := s.CreateUser("vis-pub-outsider2")
	game := Must(outsider.UploadGame("alien-first-contact"))
	Must(outsider.UpdateGame(game.ID.String(), map[string]interface{}{
		"name":   game.Name,
		"public": true,
	}))
	s.T().Logf("Outsider created public game: %s", game.ID)

	// P2 should now see the public game
	p2Games := Must(p2.ListGames())
	s.True(containsGame(p2Games, game.ID), "participant should see public games when showPublicGames is true")
	s.T().Logf("P2 sees %d games — public game correctly visible", len(p2Games))
}

// TestOwnGameAlwaysVisible verifies that a participant always sees their
// own game regardless of workshop settings.
func (s *WorkshopGameVisibilityTestSuite) TestOwnGameAlwaysVisible() {
	_, p1, _, _ := s.setupWorkshop("vis-own")

	// Both settings are false by default
	game := Must(p1.UploadGame("alien-first-contact"))

	p1Games := Must(p1.ListGames())
	s.True(containsGame(p1Games, game.ID), "participant should always see their own game")
	s.T().Logf("P1 sees own game regardless of settings")
}

// TestToggleVisibilityOnAndOff verifies that toggling showOtherParticipantsGames
// on and then off correctly shows and hides games.
func (s *WorkshopGameVisibilityTestSuite) TestToggleVisibilityOnAndOff() {
	head, p1, p2, wsIDStr := s.setupWorkshop("vis-toggle")

	game := Must(p1.UploadGame("alien-first-contact"))
	s.T().Logf("P1 created game: %s", game.ID)

	// Default: hidden
	p2Games := Must(p2.ListGames())
	s.False(containsGame(p2Games, game.ID), "should be hidden by default")

	// Enable
	ws := Must(head.GetWorkshop(wsIDStr))
	Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":                       ws.Name,
		"active":                     true,
		"showOtherParticipantsGames": true,
	}))

	p2Games = Must(p2.ListGames())
	s.True(containsGame(p2Games, game.ID), "should be visible after enabling")

	// Disable again
	Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":                       ws.Name,
		"active":                     true,
		"showOtherParticipantsGames": false,
	}))

	p2Games = Must(p2.ListGames())
	s.False(containsGame(p2Games, game.ID), "should be hidden again after disabling")
	s.T().Logf("Toggle on/off works correctly")
}

// containsGame checks if a game list contains a game with the given ID.
func containsGame(games []obj.Game, id uuid.UUID) bool {
	for _, g := range games {
		if g.ID == id {
			return true
		}
	}
	return false
}
