package testing

import (
	"fmt"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// KeySharingTestSuite checks the lean sharing model (#451) as the secure
// core decides it (core/key.go): a key is money; whoever bought it owns it,
// shares it with their own organisation and takes it back.
type KeySharingTestSuite struct {
	testutil.BaseSuite
	n int
}

func TestKeySharingSuite(t *testing.T) {
	s := &KeySharingTestSuite{}
	s.SuiteName = "Key sharing"
	suite.Run(t, s)
}

// org is an organisation at work with a head, and staff who own a key.
func (s *KeySharingTestSuite) org() (id string, head, staff *testutil.UserClient, key testutil.ApiKeyShare) {
	s.n++
	id = Must(s.DevUser().CreateInstitution(fmt.Sprintf("Schlüssel %d", s.n))).Id.String()
	member := func(role string) *testutil.UserClient {
		s.n++
		user := s.CreateUser(fmt.Sprintf("%s%d", role, s.n))
		invite := Must(s.DevUser().InviteToInstitution(id, role, user.ID))
		Must(user.AcceptInviteByToken(invite.Id.String()))
		return user
	}
	head, staff = member("head"), member("staff")
	key = Must(staff.AddApiKey("mock-key-"+fmt.Sprint(s.n), "Staff key", "mock"))
	return id, head, staff, key
}

// A key goes to its owner's own organisation only: never to another one,
// to a user or to a workshop, and only by its owner.
func (s *KeySharingTestSuite) TestAKeyStaysInItsOrganisation() {
	org, head, staff, key := s.org()
	otherOrg, _, _, _ := s.org()

	_, err := staff.ShareApiKeyWithInstitution(key.Id.String(), otherOrg)
	s.Forbidden(err, "not to another organisation")
	_, err = staff.ShareApiKeyWithUser(key.Id.String(), head.ID)
	s.BadRequest(err, "not to a user")

	_, err = head.ShareApiKeyWithInstitution(key.Id.String(), org)
	s.NotFound(err, "only the owner shares: for the head, the key's own use does not exist")

	shared := Must(staff.ShareApiKeyWithInstitution(key.Id.String(), org))
	s.Equal(org, shared.Institution.Id.String())
	s.Equal(shared.Id, Must(staff.ShareApiKeyWithInstitution(key.Id.String(), org)).Id, "sharing twice is the same share")

	listed := Must(head.GetInstitutionApiKeys(org))
	s.Len(listed, 1, "head and staff see the organisation's keys")
	s.NotEmpty(listed[0].ApiKey.KeyShortened)
}

// The owner takes a share back; a head removes a key from the organisation;
// staff who do not own it do not. The owner's own use goes with the key.
func (s *KeySharingTestSuite) TestTakingAShareBack() {
	org, head, staff, key := s.org()
	shared := Must(staff.ShareApiKeyWithInstitution(key.Id.String(), org))

	s.n++
	colleague := s.CreateUser(fmt.Sprintf("kollege%d", s.n))
	invite := Must(s.DevUser().InviteToInstitution(org, "staff", colleague.ID))
	Must(colleague.AcceptInviteByToken(invite.Id.String()))
	s.Forbidden(colleague.DeleteApiKey(shared.Id.String(), false), "staff who do not own it")

	s.NoError(head.DeleteApiKey(shared.Id.String(), false), "a head removes it from the organisation")
	s.Empty(Must(head.GetInstitutionApiKeys(org)))
	s.Len(Must(staff.GetApiKeys()).ApiKeys, 1, "the owner keeps the key")

	s.Invalid(staff.DeleteApiKey(key.Id.String(), false), "the own use goes with the key")
	s.NoError(staff.DeleteApiKey(key.Id.String(), true))
	s.Empty(Must(staff.GetApiKeys()).ApiKeys)
}

// A workshop plays with one of its organisation's keys; taking that key
// back leaves it without one.
func (s *KeySharingTestSuite) TestAWorkshopPlaysWithAnOrganisationKey() {
	org, head, staff, key := s.org()
	workshop := Must(head.CreateWorkshop(org, fmt.Sprintf("Werkstatt %d", s.n)))
	wsID := workshop.Id.String()

	own := key.Id.String()
	_, err := head.SetWorkshopApiKey(wsID, &own)
	s.Error(err, "not someone's own use of their key")

	shared := Must(staff.ShareApiKeyWithInstitution(key.Id.String(), org))
	sharedID := shared.Id.String()
	set := Must(staff.SetWorkshopApiKey(wsID, &sharedID))
	s.Equal(shared.Id, *set.DefaultApiKeyShareId)

	s.NoError(staff.DeleteApiKey(sharedID, false))
	s.Nil(Must(head.GetWorkshop(wsID)).DefaultApiKeyShareId, "no key: the workshop does not play")
}

// An organisation's free-use key is one of its own keys.
func (s *KeySharingTestSuite) TestTheFreeUseKeyIsTheOrganisations() {
	org, head, staff, key := s.org()
	own := key.Id.String()
	_, err := head.SetInstitutionFreeUseApiKey(org, &own)
	s.Error(err, "not someone's own use of their key")

	shared := Must(staff.ShareApiKeyWithInstitution(key.Id.String(), org))
	sharedID := shared.Id.String()
	set := Must(head.SetInstitutionFreeUseApiKey(org, &sharedID))
	s.Equal(shared.Id, *set.FreeUseApiKeyShareId)
}

// Nobody sees someone else's keys, the admin included.
func (s *KeySharingTestSuite) TestNobodyElseSeesAKey() {
	_, _, staff, key := s.org()
	_, err := s.DevUser().GetApiKeyInfo(key.Id.String())
	s.NotFound(err, "the admin")
	s.n++
	stranger := s.CreateUser(fmt.Sprintf("fremd%d", s.n))
	_, err = stranger.GetApiKeyInfo(key.Id.String())
	s.NotFound(err)
	s.Len(Must(staff.GetApiKeys()).ApiKeys, 1)
}
