package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ApiKeyGameSharesSuite tests the game shares visibility in API key responses
// and the GET /apikeys/{shareId}/game-shares endpoint.
type ApiKeyGameSharesSuite struct {
	testutil.BaseSuite
}

func TestApiKeyGameSharesSuite(t *testing.T) {
	s := &ApiKeyGameSharesSuite{}
	s.SuiteName = "API Key Game Shares Tests"
	suite.Run(t, s)
}

// TestPersonalGameShareVisibleInApiKeys verifies that a personal game share
// appears in the API keys response with isPrivateShare=true and correct game name.
func (s *ApiKeyGameSharesSuite) TestPersonalGameShareVisibleInApiKeys() {
	user := s.CreateUser("gs-personal")
	keyShare := Must(user.AddApiKey("mock-gs-personal", "GS Personal Key", "mock"))
	shareID := keyShare.ID.String()

	game := Must(user.UploadGame("alien-first-contact"))
	gameID := game.ID.String()

	// Create a personal game share
	created := Must(user.CreateGameShare(gameID, shareID, nil))
	s.NotEmpty(created.Token, "share should have a token")

	// Fetch API keys response - should include the game share
	resp := Must(user.GetApiKeys())
	var found bool
	for _, share := range resp.Shares {
		if share.IsPrivateShare && share.Game != nil && share.Game.ID.String() == gameID {
			found = true
			s.NotNil(share.GameShareID, "should have gameShareId set")
			s.Equal(game.Name, share.Game.Name, "game name should match")
			break
		}
	}
	s.True(found, "personal game share should appear in API keys response")
}

// TestOrgGameShareVisibleInApiKeys verifies that an org game share
// appears in the API keys response with institution context.
func (s *ApiKeyGameSharesSuite) TestOrgGameShareVisibleInApiKeys() {
	admin := s.DevUser()

	// Create institution + head
	inst := Must(admin.CreateInstitution("gs-org Org"))
	head := s.CreateUser("gs-org-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head adds API key and shares with org
	keyShare := Must(head.AddApiKey("mock-gs-org", "GS Org Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))
	orgShareID := orgShare.ID.String()

	// Upload game and create share using org key
	game := Must(head.UploadGame("alien-first-contact"))
	gameID := game.ID.String()
	Must(head.CreateGameShare(gameID, orgShareID, nil))

	// Fetch API keys - should include the org game share
	resp := Must(head.GetApiKeys())
	var found bool
	for _, share := range resp.Shares {
		if share.IsPrivateShare && share.Game != nil && share.Game.ID.String() == gameID {
			found = true
			s.NotNil(share.GameShareID, "should have gameShareId")
			s.Equal(game.Name, share.Game.Name, "game name should match")
			break
		}
	}
	s.True(found, "org game share should appear in API keys response")
}

// TestWorkshopGameShareVisibleInApiKeys verifies that a workshop game share
// appears in the API keys response.
func (s *ApiKeyGameSharesSuite) TestWorkshopGameShareVisibleInApiKeys() {
	admin := s.DevUser()

	// Create institution + head
	inst := Must(admin.CreateInstitution("gs-ws Org"))
	head := s.CreateUser("gs-ws-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	// Head adds API key, shares with org
	keyShare := Must(head.AddApiKey("mock-gs-ws", "GS WS Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))

	// Create workshop and set API key
	workshop := Must(head.CreateWorkshop(inst.ID.String(), "GS Workshop"))
	wsIDStr := workshop.ID.String()
	orgShareIDStr := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsIDStr, &orgShareIDStr))

	Must(head.UpdateWorkshop(wsIDStr, map[string]interface{}{
		"name":             "GS Workshop",
		"active":           true,
		"public":           false,
		"allowGameSharing": true,
		"isPaused":         false,
	}))

	// Participant joins and creates workshop share
	invite := Must(head.CreateWorkshopInvite(wsIDStr, string(obj.RoleParticipant)))
	s.Require().NotNil(invite.InviteToken)

	participant := s.CreateUser("gs-ws-participant")
	s.Require().NoError(participant.AcceptWorkshopInviteByToken(*invite.InviteToken))
	game := Must(participant.UploadGame("alien-first-contact"))
	gameID := game.ID.String()
	Must(participant.CreateWorkshopGameShare(gameID, wsIDStr, nil))

	// Head should see the workshop game share in API keys response
	// (the key belongs to head, so game share is under head's key)
	resp := Must(head.GetApiKeys())
	var found bool
	for _, share := range resp.Shares {
		if share.IsPrivateShare && share.Game != nil && share.Game.ID.String() == gameID {
			found = true
			break
		}
	}
	s.True(found, "workshop game share should appear in head's API keys response")
}

// TestGameSharesEndpointPersonalContext verifies the personal context returns
// all shares for the key owner.
func (s *ApiKeyGameSharesSuite) TestGameSharesEndpointPersonalContext() {
	user := s.CreateUser("gs-ctx-personal")
	keyShare := Must(user.AddApiKey("mock-gs-ctx-personal", "CTX Personal Key", "mock"))
	shareID := keyShare.ID.String()

	game := Must(user.UploadGame("alien-first-contact"))
	gameID := game.ID.String()

	Must(user.CreateGameShare(gameID, shareID, nil))

	// Personal context - owner should see all shares
	gameShares := Must(user.GetApiKeyGameShares(shareID, "personal"))
	s.Require().Len(gameShares, 1, "owner should see personal share in personal context")
	s.Equal("personal", gameShares[0].Source)
	s.Equal(game.Name, gameShares[0].GameName)
	s.Equal(gameID, gameShares[0].GameID.String())
}

// TestGameSharesEndpointOrgContextFiltersPersonal verifies the organization context
// excludes personal shares.
func (s *ApiKeyGameSharesSuite) TestGameSharesEndpointOrgContextFiltersPersonal() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("gs-ctx-filter Org"))
	head := s.CreateUser("gs-ctx-filter-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	keyShare := Must(head.AddApiKey("mock-gs-ctx-filter", "CTX Filter Key", "mock"))
	selfShareID := keyShare.ID.String()
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))
	orgShareID := orgShare.ID.String()

	game := Must(head.UploadGame("alien-first-contact"))
	gameID := game.ID.String()

	// Create both a personal and an org share for the same game
	Must(head.CreateGameShare(gameID, selfShareID, nil))
	Must(head.CreateGameShare(gameID, orgShareID, nil))

	// Personal context (owner) - should see both
	personalShares := Must(head.GetApiKeyGameShares(selfShareID, "personal"))
	s.Require().Len(personalShares, 2, "owner should see both shares in personal context")

	// Org context - should only see the org share
	orgShares := Must(head.GetApiKeyGameShares(orgShareID, "organization"))
	s.Require().Len(orgShares, 1, "org context should only show org shares")
	s.Equal("organization", orgShares[0].Source, "filtered share should be organization")
}

// TestGameSharesEndpointPersonalContextDeniedForNonOwner verifies that
// non-owners cannot use personal context.
func (s *ApiKeyGameSharesSuite) TestGameSharesEndpointPersonalContextDeniedForNonOwner() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("gs-ctx-deny Org"))
	head := s.CreateUser("gs-ctx-deny-head")
	staff := s.CreateUser("gs-ctx-deny-staff")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))
	staffInvite := Must(admin.InviteToInstitution(inst.ID.String(), "staff", staff.ID))
	Must(staff.AcceptInvite(staffInvite.ID.String()))

	keyShare := Must(head.AddApiKey("mock-gs-ctx-deny", "CTX Deny Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))
	orgShareID := orgShare.ID.String()

	game := Must(head.UploadGame("alien-first-contact"))
	Must(head.CreateGameShare(game.ID.String(), orgShareID, nil))

	// Staff can access org context (they have access via institution)
	orgShares := Must(staff.GetApiKeyGameShares(orgShareID, "organization"))
	s.Require().Len(orgShares, 1, "staff should see org shares in organization context")

	// Staff cannot use personal context (not key owner)
	_, err := staff.GetApiKeyGameShares(orgShareID, "personal")
	s.Error(err, "non-owner should be denied personal context")
}

// TestGameSharesEndpointWithRemaining verifies the endpoint returns remaining session count.
func (s *ApiKeyGameSharesSuite) TestGameSharesEndpointWithRemaining() {
	user := s.CreateUser("gs-remaining")
	keyShare := Must(user.AddApiKey("mock-gs-remaining", "GS Remaining Key", "mock"))
	shareID := keyShare.ID.String()

	game := Must(user.UploadGame("alien-first-contact"))
	gameID := game.ID.String()

	// Create a game share with max sessions
	maxSessions := 5
	Must(user.CreateGameShare(gameID, shareID, &maxSessions))

	gameShares := Must(user.GetApiKeyGameShares(shareID, "personal"))
	s.Require().Len(gameShares, 1)
	s.NotNil(gameShares[0].Remaining, "remaining should be set")
	s.Equal(5, *gameShares[0].Remaining, "remaining should be 5")
}

// TestGameSharesEndpointPermission verifies that a user cannot access
// another user's key game shares at all.
func (s *ApiKeyGameSharesSuite) TestGameSharesEndpointPermission() {
	user1 := s.CreateUser("gs-perm-owner")
	user2 := s.CreateUser("gs-perm-other")

	keyShare := Must(user1.AddApiKey("mock-gs-perm", "GS Perm Key", "mock"))
	shareID := keyShare.ID.String()

	game := Must(user1.UploadGame("alien-first-contact"))
	Must(user1.CreateGameShare(game.ID.String(), shareID, nil))

	// user2 should not be able to access user1's game shares (no context)
	_, err := user2.GetApiKeyGameShares(shareID, "")
	s.Error(err, "non-owner should not be able to access game shares")
}

// TestGameSharesEndpointOrgSource verifies the endpoint returns "organization"
// source for org-shared key game shares.
func (s *ApiKeyGameSharesSuite) TestGameSharesEndpointOrgSource() {
	admin := s.DevUser()

	inst := Must(admin.CreateInstitution("gs-ep-org Org"))
	head := s.CreateUser("gs-ep-org-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	keyShare := Must(head.AddApiKey("mock-gs-ep-org", "GS EP Org Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))
	orgShareID := orgShare.ID.String()

	game := Must(head.UploadGame("alien-first-contact"))
	Must(head.CreateGameShare(game.ID.String(), orgShareID, nil))

	// No context filter - should return all
	gameShares := Must(head.GetApiKeyGameShares(orgShareID, ""))
	s.Require().Len(gameShares, 1)
	s.Equal("organization", gameShares[0].Source, "source should be organization")
	s.Equal(game.Name, gameShares[0].GameName)
}
