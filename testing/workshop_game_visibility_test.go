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

// TestHeadAlwaysSeesAllParticipantGamesRegardlessOfSettings verifies that a head
// in workshop mode sees all participants' games even when showOtherParticipantsGames=false.
func (s *WorkshopGameVisibilityTestSuite) TestHeadAlwaysSeesAllParticipantGamesRegardlessOfSettings() {
	head, p1, p2, wsIDStr := s.setupWorkshop("head-bypass")

	// Head enters workshop mode (required to see workshop games in game list)
	Must(head.SetActiveWorkshop(&wsIDStr))
	s.T().Logf("Head entered workshop mode (showOtherParticipantsGames=false by default)")

	// Both participants create a game
	game1 := Must(p1.UploadGame("alien-first-contact"))
	game2 := Must(p2.UploadGame("alien-first-contact"))
	s.T().Logf("P1 game: %s, P2 game: %s", game1.ID, game2.ID)

	// Head should see BOTH games even though showOtherParticipantsGames=false
	headGames := Must(head.ListGames())
	s.True(containsGame(headGames, game1.ID), "head should see p1's game regardless of showOtherParticipantsGames")
	s.True(containsGame(headGames, game2.ID), "head should see p2's game regardless of showOtherParticipantsGames")
	s.T().Logf("Head sees %d games — both visible despite showOtherParticipantsGames=false", len(headGames))
}

// TestHeadRespectsShowPublicGamesSetting verifies that a head
// in workshop mode does NOT see public games from outside when showPublicGames=false,
// but DOES see them when showPublicGames=true. The setting applies to all roles.
func (s *WorkshopGameVisibilityTestSuite) TestHeadRespectsShowPublicGamesSetting() {
	head, _, _, wsIDStr := s.setupWorkshop("head-pub-setting")

	// Head enters workshop mode (showPublicGames stays false by default)
	Must(head.SetActiveWorkshop(&wsIDStr))
	s.T().Logf("Head entered workshop mode (showPublicGames=false by default)")

	// Create a public game from an outside user
	outsider := s.CreateUser("head-pub-outsider")
	game := Must(outsider.UploadGame("alien-first-contact"))
	Must(outsider.UpdateGame(game.ID.String(), map[string]interface{}{
		"name":   game.Name,
		"public": true,
	}))
	s.T().Logf("Outsider public game: %s", game.ID)

	// Head should NOT see the public game when showPublicGames=false
	headGames := Must(head.ListGames())
	s.False(containsGame(headGames, game.ID), "head should not see public games when showPublicGames=false")
	s.T().Logf("Head sees %d games — public game hidden with showPublicGames=false", len(headGames))

	// Enable showPublicGames and verify head now sees the public game
	Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":                       "head-pub-setting Workshop",
		"active":                     true,
		"showPublicGames":            true,
		"showOtherParticipantsGames": true,
	}))
	headGames = Must(head.ListGames())
	s.True(containsGame(headGames, game.ID), "head should see public games when showPublicGames=true")
	s.T().Logf("Head sees %d games — public game visible with showPublicGames=true", len(headGames))
}

// TestStaffAlwaysSeesAllParticipantGamesRegardlessOfSettings verifies that staff
// in workshop mode see all participants' games even when showOtherParticipantsGames=false.
func (s *WorkshopGameVisibilityTestSuite) TestStaffAlwaysSeesAllParticipantGamesRegardlessOfSettings() {
	head, p1, _, wsIDStr := s.setupWorkshop("staff-bypass")

	// Create a staff member and add them to the same institution
	headUser := Must(head.GetMe())
	instID := ""
	if headUser.Role != nil && headUser.Role.Institution != nil {
		instID = headUser.Role.Institution.ID.String()
	}
	s.Require().NotEmpty(instID, "head should have an institution")

	staff := s.CreateUser("staff-bypass-member")
	staffInvite := Must(head.InviteToInstitution(instID, "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))
	s.T().Logf("Staff member joined institution")

	// Staff enters workshop mode
	Must(staff.SetActiveWorkshop(&wsIDStr))
	s.T().Logf("Staff entered workshop mode (showOtherParticipantsGames=false by default)")

	// P1 creates a game
	game1 := Must(p1.UploadGame("alien-first-contact"))
	s.T().Logf("P1 game: %s", game1.ID)

	// Staff should see p1's game even though showOtherParticipantsGames=false
	staffGames := Must(staff.ListGames())
	s.True(containsGame(staffGames, game1.ID), "staff should see participant's game regardless of showOtherParticipantsGames")
	s.T().Logf("Staff sees %d games — visible despite showOtherParticipantsGames=false", len(staffGames))
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
