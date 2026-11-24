package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// NotificationsTestSuite checks the bell (#578): important facts that are
// true now and wait for the user's acknowledgement. An entry goes when it
// is acknowledged or no longer true; read state is personal, and a user's
// starting point is when they got their role.
type NotificationsTestSuite struct {
	testutil.BaseSuite
}

func TestNotificationsSuite(t *testing.T) {
	s := &NotificationsTestSuite{}
	s.SuiteName = "Notifications"
	suite.Run(t, s)
}

// newAdmin is an admin who got the role just now.
func (s *NotificationsTestSuite) newAdmin() *testutil.UserClient {
	admin := s.CreateUser(testutil.Unique("admin"))
	Must(s.DevUser().SetUserRole(admin.ID, "admin"))
	return admin
}

// organisation is an organisation the dev admin created, with a head and a
// member of staff.
type organisation struct {
	id          string
	head, staff *testutil.UserClient
}

func (s *NotificationsTestSuite) organisation() organisation {
	id := Must(s.DevUser().CreateInstitution(testutil.Unique("Glocke"))).Id.String()
	return organisation{id: id, head: s.member(id, "head"), staff: s.member(id, "staff")}
}

// member is a new account that joined organisation org in role.
func (s *NotificationsTestSuite) member(org, role string) *testutil.UserClient {
	u := s.CreateUser(testutil.Unique(role))
	Must(u.AcceptInviteByToken(Must(s.DevUser().InviteToInstitution(org, role, u.ID)).Id.String()))
	return u
}

// requestPage registers organisation o's verified page, waiting for approval.
func (s *NotificationsTestSuite) requestPage(o organisation) testutil.VerifiedPage {
	return Must(o.head.CreateVerifiedPage(o.id, "http://127.0.0.1:9/play", "http://127.0.0.1:9/callback"))
}

// A new admin is told about the organisations created since they became
// admin, folded into one entry with their number; older ones are no news.
func (s *NotificationsTestSuite) TestANewAdminHearsOfOrganisationsCreatedSinceTheirRoleStarted() {
	Must(s.DevUser().CreateInstitution(testutil.Unique("Vorher")))
	admin := s.newAdmin()
	s.Nil(Must(admin.Notification(testutil.NewOrganisations)))

	Must(s.DevUser().CreateInstitution(testutil.Unique("Neu")))
	Must(s.DevUser().CreateInstitution(testutil.Unique("Neu")))

	n := Must(admin.Notification(testutil.NewOrganisations))
	s.Require().NotNil(n)
	s.Equal(2, n.Count)
}

// Acknowledging clears the entry for the one admin only; the next new
// organisation starts a fresh entry.
func (s *NotificationsTestSuite) TestAcknowledgingIsPersonalAndTheNextFactStartsAFreshEntry() {
	admin, other := s.newAdmin(), s.newAdmin()
	Must(s.DevUser().CreateInstitution(testutil.Unique("Neu")))
	Must(s.DevUser().CreateInstitution(testutil.Unique("Neu")))

	MustSucceed(admin.Acknowledge(*Must(admin.Notification(testutil.NewOrganisations))))
	s.Nil(Must(admin.Notification(testutil.NewOrganisations)))
	s.Equal(2, Must(other.Notification(testutil.NewOrganisations)).Count, "another admin's entry stays")

	Must(s.DevUser().CreateInstitution(testutil.Unique("Danach")))
	n := Must(admin.Notification(testutil.NewOrganisations))
	s.Require().NotNil(n)
	s.Equal(1, n.Count)
}

// An organisation deleted before anyone looked is no news any more; an
// admin is not told about the organisations they created themselves.
func (s *NotificationsTestSuite) TestNewOrganisationsAreThoseThatStillExistAndOthersCreated() {
	admin := s.newAdmin()
	gone := Must(s.DevUser().CreateInstitution(testutil.Unique("Weg"))).Id.String()
	Must(admin.CreateInstitution(testutil.Unique("Eigene")))
	s.Equal(1, Must(admin.Notification(testutil.NewOrganisations)).Count)

	MustSucceed(s.DevUser().DeleteInstitution(gone))
	s.Nil(Must(admin.Notification(testutil.NewOrganisations)))
}

// The pages waiting for approval count down as admins decide them, as
// they are removed and as their organisations are deleted; at none the
// entry goes.
func (s *NotificationsTestSuite) TestPagesWaitingCountDownAsTheyAreDecided() {
	admin := s.newAdmin()
	approved, refused, removed, orgGone := s.organisation(), s.organisation(), s.organisation(), s.organisation()
	pages := map[string]testutil.VerifiedPage{}
	for name, o := range map[string]organisation{"approved": approved, "refused": refused, "removed": removed, "orgGone": orgGone} {
		pages[name] = s.requestPage(o)
	}
	waiting := func() int {
		n := Must(admin.Notification(testutil.PagesWaiting))
		if n == nil {
			return 0
		}
		return n.Count
	}
	s.Equal(4, waiting())

	Must(s.DevUser().ApproveVerifiedPage(pages["approved"].Id.String(), true))
	s.Equal(3, waiting())
	Must(s.DevUser().ApproveVerifiedPage(pages["refused"].Id.String(), false))
	s.Equal(2, waiting())
	MustSucceed(removed.head.DeleteVerifiedPage(pages["removed"].Id.String()))
	s.Equal(1, waiting())
	MustSucceed(s.DevUser().DeleteInstitution(orgGone.id))
	s.Equal(0, waiting())
}

// An admin who acknowledged the waiting pages hears again only of pages
// asked for since, a page asked for anew after a change of site among them.
func (s *NotificationsTestSuite) TestPagesWaitingAreThoseAskedForSinceTheAcknowledgement() {
	admin := s.newAdmin()
	early, late := s.organisation(), s.organisation()
	page := s.requestPage(early)
	MustSucceed(admin.Acknowledge(*Must(admin.Notification(testutil.PagesWaiting))))
	s.Nil(Must(admin.Notification(testutil.PagesWaiting)))

	s.requestPage(late)
	s.Equal(1, Must(admin.Notification(testutil.PagesWaiting)).Count)

	Must(early.head.UpdateVerifiedPage(page.Id.String(), "http://localhost:9/play", "http://localhost:9/callback"))
	s.Equal(2, Must(admin.Notification(testutil.PagesWaiting)).Count)
}

// An organisation's head and staff hear that its page was approved, each
// acknowledging for themselves; staff who joined later are not told.
func (s *NotificationsTestSuite) TestAnOrganisationHearsItsPageWasApproved() {
	o := s.organisation()
	page := s.requestPage(o)
	s.Nil(Must(o.head.Notification(testutil.PageApproved)))

	Must(s.DevUser().ApproveVerifiedPage(page.Id.String(), true))
	n := Must(o.head.Notification(testutil.PageApproved))
	s.Require().NotNil(n)
	s.Require().NotNil(n.OrganisationId)
	s.Equal(o.id, n.OrganisationId.String())
	s.NotNil(Must(o.staff.Notification(testutil.PageApproved)))
	s.Nil(Must(s.member(o.id, "staff").Notification(testutil.PageApproved)), "staff who joined after the approval")

	MustSucceed(o.head.Acknowledge(*n))
	s.Nil(Must(o.head.Notification(testutil.PageApproved)))
	s.NotNil(Must(o.staff.Notification(testutil.PageApproved)))
}

// An organisation hears that an admin turned its page down or withdrew its
// approval; an approval lost to its own change of site is no news, and a
// new approval or a new request replaces the refusal.
func (s *NotificationsTestSuite) TestAnOrganisationHearsItsPageWasTurnedDown() {
	o := s.organisation()
	page := s.requestPage(o).Id.String()

	Must(s.DevUser().ApproveVerifiedPage(page, false))
	s.NotNil(Must(o.staff.Notification(testutil.PageTurnedDown)))
	Must(o.head.UpdateVerifiedPage(page, "http://localhost:9/play", "http://localhost:9/callback"))
	s.Nil(Must(o.staff.Notification(testutil.PageTurnedDown)), "asked for anew")

	Must(s.DevUser().ApproveVerifiedPage(page, true))
	s.Nil(Must(o.head.Notification(testutil.PageTurnedDown)))
	s.NotNil(Must(o.head.Notification(testutil.PageApproved)))

	Must(s.DevUser().ApproveVerifiedPage(page, false))
	s.NotNil(Must(o.head.Notification(testutil.PageTurnedDown)), "approval withdrawn")
	s.Nil(Must(o.head.Notification(testutil.PageApproved)))

	MustSucceed(o.head.AcknowledgeAll())
	Must(s.DevUser().ApproveVerifiedPage(page, true))
	MustSucceed(o.head.AcknowledgeAll())
	Must(o.head.UpdateVerifiedPage(page, "http://127.0.0.1:9/play", "http://127.0.0.1:9/callback"))
	s.Empty(Must(o.head.Notifications()), "its own change of site withdrew the approval")
}

// Acknowledging all clears every entry the caller has.
func (s *NotificationsTestSuite) TestAcknowledgingAllClearsEveryEntry() {
	admin := s.newAdmin()
	s.requestPage(s.organisation())
	s.Len(Must(admin.Notifications()), 2, "a new organisation and its page")

	MustSucceed(admin.AcknowledgeAll())
	s.Empty(Must(admin.Notifications()))
}

// Notifications are the caller's own: whoever holds no type has none, and
// acknowledging one they cannot have is refused.
func (s *NotificationsTestSuite) TestNotificationsAreOnlyTheCallersOwn() {
	o := s.organisation()
	s.requestPage(o)
	workshop := Must(o.head.CreateWorkshop(o.id, testutil.Unique("Werkstatt"))).Id.String()
	guest := Must(s.JoinWorkshop(*Must(o.head.CreateWorkshopInvite(workshop)).InviteToken))
	individual := s.CreateUser(testutil.Unique("individual"))

	s.Empty(Must(guest.Notifications()))
	s.Empty(Must(individual.Notifications()))
	s.Empty(Must(o.head.Notifications()))
	s.NotFound(o.head.AcknowledgeType(testutil.NewOrganisations))
	s.NotFound(guest.AcknowledgeType(testutil.PagesWaiting))
}
