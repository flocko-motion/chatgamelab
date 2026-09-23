package testing

import (
	"cgl/obj"
	"cgl/testing/testutil"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// WorkshopWordTokensTestSuite tests invite and re-login tokens made of words:
// format, tolerant input, participant login without prefix, access reset, and
// replacement of a used-up invite.
type WorkshopWordTokensTestSuite struct {
	testutil.BaseSuite
}

func TestWorkshopWordTokensSuite(t *testing.T) {
	s := &WorkshopWordTokensTestSuite{}
	s.SuiteName = "Workshop Word Tokens Tests"
	suite.Run(t, s)
}

var (
	threeWords = regexp.MustCompile(`^[a-z]{4,10}(-[a-z]{4,10}){2}$`)
	fourWords  = regexp.MustCompile(`^participant-[a-z]{4,10}(-[a-z]{4,10}){3}$`)
)

func (s *WorkshopWordTokensTestSuite) workshopSetup(prefix string) (*testutil.UserClient, string) {
	admin := s.DevUser()
	inst := Must(admin.CreateInstitution(prefix + " Org"))
	head := s.CreateUser(prefix + "-head")
	headInvite := Must(admin.InviteToInstitution(inst.ID.String(), "head", head.ID))
	Must(head.AcceptInvite(headInvite.ID.String()))

	keyShare := Must(head.AddApiKey("mock-word-key-"+prefix, prefix+" Key", "mock"))
	orgShare := Must(head.ShareApiKeyWithInstitution(keyShare.ID.String(), inst.ID.String()))
	workshop := Must(head.CreateWorkshop(inst.ID.String(), prefix+" Workshop"))
	wsID := workshop.ID.String()
	orgShareID := orgShare.ID.String()
	Must(head.SetWorkshopApiKey(wsID, &orgShareID))
	Must(head.UpdateWorkshop(wsID, map[string]interface{}{
		"name":     prefix + " Workshop",
		"active":   true,
		"public":   false,
		"isPaused": false,
	}))
	return head, wsID
}

func (s *WorkshopWordTokensTestSuite) TestInviteTokenIsThreeWords() {
	head, wsID := s.workshopSetup("word-invite")
	invite := Must(head.CreateWorkshopInvite(wsID, string(obj.RoleParticipant)))
	s.Require().NotNil(invite.InviteToken)
	s.Regexp(threeWords, *invite.InviteToken)

	// Typed with capitals and spaces instead of hyphens.
	typed := strings.ToUpper(strings.ReplaceAll(*invite.InviteToken, "-", " "))
	var found map[string]interface{}
	s.NoError(s.Public().Get("invites/"+url.PathEscape(typed), &found))

	resp := Must(s.AcceptWorkshopInviteAnonymously(url.PathEscape(typed)))
	s.Require().NotNil(resp.AuthToken)
	s.Regexp(fourWords, *resp.AuthToken)
}

func (s *WorkshopWordTokensTestSuite) TestParticipantLoginWithoutPrefix() {
	head, wsID := s.workshopSetup("word-login")
	invite := Must(head.CreateWorkshopInvite(wsID, string(obj.RoleParticipant)))
	resp := Must(s.AcceptWorkshopInviteAnonymously(*invite.InviteToken))
	words := strings.TrimPrefix(*resp.AuthToken, "participant-")

	s.NoError(s.Public().Post("auth/participant-login", map[string]string{"token": strings.ToUpper(words)}, nil))
	s.Error(s.Public().Post("auth/participant-login", map[string]string{"token": "apfel-otter-turm-leise"}, nil))
}

func (s *WorkshopWordTokensTestSuite) TestResetParticipantToken() {
	head, wsID := s.workshopSetup("word-reset")
	invite := Must(head.CreateWorkshopInvite(wsID, string(obj.RoleParticipant)))
	resp := Must(s.AcceptWorkshopInviteAnonymously(*invite.InviteToken))
	participant := s.CreateUserWithToken(*resp.AuthToken)

	var result map[string]string
	s.Require().NoError(head.Post("workshops/participants/"+participant.ID+"/token/reset", nil, &result))
	newToken := result["token"]
	s.Regexp(fourWords, newToken)
	s.NotEqual(*resp.AuthToken, newToken)

	var me obj.User
	s.Error(participant.Get("users/me", &me), "old token must stop working")
	s.NotNil(s.CreateUserWithToken(newToken))

	// A participant cannot reset anyone's token.
	s.Error(s.CreateUserWithToken(newToken).Post("workshops/participants/"+participant.ID+"/token/reset", nil, nil))
}

func (s *WorkshopWordTokensTestSuite) TestUsedUpInviteIsReplaced() {
	head, wsID := s.workshopSetup("word-usedup")
	var first obj.UserRoleInvite
	s.Require().NoError(head.Post("invites/workshop", map[string]interface{}{
		"workshopId": wsID,
		"maxUses":    1,
	}, &first))
	Must(s.AcceptWorkshopInviteAnonymously(*first.InviteToken))

	second := Must(head.CreateWorkshopInvite(wsID, string(obj.RoleParticipant)))
	s.NotEqual(first.ID, second.ID, "a used-up invite must be replaced")
	s.Equal(obj.InviteStatusPending, second.Status)
}
