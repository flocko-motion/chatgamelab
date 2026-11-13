package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"testing"

	"github.com/stretchr/testify/suite"
)

// AiQualityTierLimitTestSuite tests the release of free-use keys: a free-use key
// (the organisation's or the server's) plays at the tier asked for — the
// workshop's in a workshop, else the user's own — but at most at the tier it is
// released up to. A key of one's own (workshop or personal) plays at any tier.
type AiQualityTierLimitTestSuite struct {
	testutil.BaseSuite
}

func TestAiQualityTierLimitSuite(t *testing.T) {
	s := &AiQualityTierLimitTestSuite{}
	s.SuiteName = "AI Quality Tier Limit Tests"
	suite.Run(t, s)
}

type tierOrg struct {
	instID string
	head   *testutil.UserClient
	keyID  string // the org-shared key's share ID
}

// org creates an institution with a head whose mock key is the organisation's
// free-use key, released up to release ("" = server default).
func (s *AiQualityTierLimitTestSuite) org(prefix, release string) tierOrg {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	invite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.ID.String()))

	key := Must(head.AddApiKey("mock-"+prefix, prefix+" Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(key.ID.String(), inst.ID.String()))
	shareID := orgShare.ID.String()
	Must(head.SetInstitutionFreeUseApiKey(inst.ID.String(), &shareID))
	if release != "" {
		Must(head.SetInstitutionFreeUseTier(inst.ID.String(), release))
	}
	return tierOrg{instID: inst.ID.String(), head: head, keyID: shareID}
}

// workshop creates a workshop with the given tier and a participant in it.
func (s *AiQualityTierLimitTestSuite) workshop(o tierOrg, name, tier string) (obj.Workshop, *testutil.UserClient) {
	ws := Must(o.head.CreateWorkshop(o.instID, name))
	ws = Must(o.head.UpdateWorkshop(ws.ID.String(), map[string]interface{}{
		"name": name, "active": true, "public": false, "aiQualityTier": tier,
	}))
	invite := Must(o.head.CreateWorkshopInvite(ws.ID.String(), string(obj.RoleParticipant)))
	resp := Must(s.AcceptWorkshopInviteAnonymously(*invite.InviteToken))
	return ws, s.CreateUserWithToken(*resp.AuthToken)
}

func (s *AiQualityTierLimitTestSuite) TestWorkshopWithoutKeyIsCappedAtOrganisationRelease() {
	o := s.org("tier-ws-cap", obj.AiModelBalanced)
	ws, participant := s.workshop(o, "Tier Cap Workshop", obj.AiModelMax)

	ws = Must(o.head.GetWorkshop(ws.ID.String()))
	s.Equal(obj.WorkshopKeySourceOrganization, ws.ApiKeySource)
	s.Require().NotNil(ws.AiQualityTierLimit)
	s.Equal(obj.AiModelBalanced, *ws.AiQualityTierLimit)

	game := Must(participant.UploadGame("alien-first-contact"))
	session := Must(participant.CreateGameSession(game.ID.String()))
	s.Equal(obj.AiModelBalanced, session.AiModel, "the organisation's key is released only up to medium")
	s.Equal(obj.AiModelMax, session.AiQualityTierRequested)
	s.Equal(obj.AiQualityTierLimitedByOrganization, session.AiQualityTierLimitedBy)

	stored := Must(participant.GetGameSession(session.ID.String()))
	s.Equal(obj.AiModelBalanced, stored.AiModel)
	s.Equal(obj.AiModelMax, stored.AiQualityTierRequested, "the cap is stored, so the player can say why")
	s.Equal(obj.AiQualityTierLimitedByOrganization, stored.AiQualityTierLimitedBy)
}

func (s *AiQualityTierLimitTestSuite) TestWorkshopTierBelowReleaseApplies() {
	o := s.org("tier-ws-low", obj.AiModelPremium)
	_, participant := s.workshop(o, "Tier Low Workshop", obj.AiModelEconomy)

	game := Must(participant.UploadGame("alien-first-contact"))
	session := Must(participant.CreateGameSession(game.ID.String()))
	s.Equal(obj.AiModelEconomy, session.AiModel, "a tier below the release is used as chosen")
	s.Empty(session.AiQualityTierRequested)
	s.Empty(session.AiQualityTierLimitedBy)
}

func (s *AiQualityTierLimitTestSuite) TestWorkshopWithoutTierPlaysAtRelease() {
	o := s.org("tier-ws-none", obj.AiModelPremium)
	ws := Must(o.head.CreateWorkshop(o.instID, "Tier None Workshop"))
	invite := Must(o.head.CreateWorkshopInvite(ws.ID.String(), string(obj.RoleParticipant)))
	resp := Must(s.AcceptWorkshopInviteAnonymously(*invite.InviteToken))
	participant := s.CreateUserWithToken(*resp.AuthToken)

	game := Must(participant.UploadGame("alien-first-contact"))
	session := Must(participant.CreateGameSession(game.ID.String()))
	s.Equal(obj.AiModelPremium, session.AiModel, "without a tier of its own the workshop plays at the release")
	s.Empty(session.AiQualityTierRequested)
}

func (s *AiQualityTierLimitTestSuite) TestWorkshopOwnKeyIsNotCapped() {
	o := s.org("tier-ws-own", obj.AiModelEconomy)
	ws, participant := s.workshop(o, "Tier Own Key Workshop", obj.AiModelPremium)
	Must(o.head.SetWorkshopApiKey(ws.ID.String(), &o.keyID))

	ws = Must(o.head.GetWorkshop(ws.ID.String()))
	s.Equal(obj.WorkshopKeySourceWorkshop, ws.ApiKeySource)
	s.Nil(ws.AiQualityTierLimit, "a workshop's own key has no limit")

	game := Must(participant.UploadGame("alien-first-contact"))
	session := Must(participant.CreateGameSession(game.ID.String()))
	s.Equal(obj.AiModelPremium, session.AiModel)
	s.Empty(session.AiQualityTierRequested)
}

func (s *AiQualityTierLimitTestSuite) TestMemberIsCappedAtOrganisationRelease() {
	o := s.org("tier-member", obj.AiModelPremium)
	staff := s.CreateUser("tier-member-staff")
	invite := Must(o.head.InviteToInstitution(o.instID, "staff", staff.ID))
	Must(staff.AcceptInvite(invite.ID.String()))
	s.Require().NoError(staff.SetUserAiQualityTier(obj.AiModelMax))

	me := Must(staff.GetMe())
	s.Equal(obj.UserTierSourceOrganization, me.AiQualityTierLimitSource)
	s.Require().NotNil(me.AiQualityTierLimit)
	s.Equal(obj.AiModelPremium, *me.AiQualityTierLimit)

	game := Must(staff.UploadGame("alien-first-contact"))
	session := Must(staff.CreateGameSession(game.ID.String()))
	s.Equal(obj.AiModelPremium, session.AiModel)
	s.Equal(obj.AiModelMax, session.AiQualityTierRequested)
	s.Equal(obj.AiQualityTierLimitedByOrganization, session.AiQualityTierLimitedBy)
}

func (s *AiQualityTierLimitTestSuite) TestOwnKeyIsNotCapped() {
	user := s.CreateUser("tier-own-key")
	key := Must(user.AddApiKey("mock-tier-own", "Own Key", "mock"))
	Must(user.SetDefaultApiKey(key.ID.String()))
	s.Require().NoError(user.SetUserAiQualityTier(obj.AiModelPremium))

	me := Must(user.GetMe())
	s.Equal(obj.UserTierSourceOwn, me.AiQualityTierLimitSource)
	s.Nil(me.AiQualityTierLimit)

	game := Must(user.UploadGame("alien-first-contact"))
	session := Must(user.CreateGameSession(game.ID.String()))
	s.Equal(obj.AiModelPremium, session.AiModel)
	s.Empty(session.AiQualityTierRequested)
}

func (s *AiQualityTierLimitTestSuite) TestServerFreeKeyIsCappedAtServerRelease() {
	admin := s.DevUser()
	key := Must(admin.AddApiKey("mock-tier-server", "Server Free Key", "mock"))
	keyID := key.ApiKeyID.String()
	Must(admin.SetSystemFreeUseApiKey(&keyID))
	Must(admin.SetSystemFreeUseTier(obj.AiModelEconomy))
	defer func() {
		Must(admin.SetSystemFreeUseTier(""))
		Must(admin.SetSystemFreeUseApiKey(nil))
	}()

	user := s.CreateUser("tier-server-user")
	s.Require().NoError(user.SetUserAiQualityTier(obj.AiModelPremium))

	me := Must(user.GetMe())
	s.Equal(obj.UserTierSourceServer, me.AiQualityTierLimitSource)
	s.Require().NotNil(me.AiQualityTierLimit)
	s.Equal(obj.AiModelEconomy, *me.AiQualityTierLimit)

	game := Must(user.UploadGame("alien-first-contact"))
	session := Must(user.CreateGameSession(game.ID.String()))
	s.Equal(obj.AiModelEconomy, session.AiModel)
	s.Equal(obj.AiModelPremium, session.AiQualityTierRequested)
	s.Equal(obj.AiQualityTierLimitedByServer, session.AiQualityTierLimitedBy)
}

func (s *AiQualityTierLimitTestSuite) TestWhoSetsOrganisationRelease() {
	o := s.org("tier-perm", "")
	staff := s.CreateUser("tier-perm-staff")
	invite := Must(o.head.InviteToInstitution(o.instID, "staff", staff.ID))
	Must(staff.AcceptInvite(invite.ID.String()))
	_, participant := s.workshop(o, "Tier Perm Workshop", "")

	// Like the free-use key itself, its release is set by those who manage the organisation.
	Must(staff.SetInstitutionFreeUseTier(o.instID, obj.AiModelMax))
	_, err := participant.SetInstitutionFreeUseTier(o.instID, obj.AiModelEconomy)
	s.Error(err, "participants may not change the release")
	s.Contains(err.Error(), "403")

	_, err = o.head.SetInstitutionFreeUseTier(o.instID, "ultra")
	s.Error(err, "an unknown tier is refused")
	s.Contains(err.Error(), "400")

	inst := Must(o.head.SetInstitutionFreeUseTier(o.instID, obj.AiModelPremium))
	s.Require().NotNil(inst.FreeUseAiQualityTier)
	s.Equal(obj.AiModelPremium, *inst.FreeUseAiQualityTier)

	inst = Must(o.head.SetInstitutionFreeUseTier(o.instID, ""))
	s.Nil(inst.FreeUseAiQualityTier, "an empty tier falls back to the server default")
}
