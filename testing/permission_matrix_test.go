package testing

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// The permission matrix (#344): for every kind of caller and every fetch and
// escalation, whether it is granted on that caller's own resources and
// refused on someone else's. It is written by hand and is the specification:
// a rule change shows here in the same review. It is not derived from the
// grant table (core/internal/rules), which it checks.
//
// The callers of one world:
//
//	admin   the site's admin
//	headA   head of organisation A, staffA its staff
//	guestA  a guest in workshop A (organisation A's)
//	indivA  an individual taking part in workshop A, through its invite
//	indiv   an individual taking part in nothing
//	headB   head of organisation B, with workshop B and its guest
//	visitor nobody signed in
type PermissionMatrixTestSuite struct {
	testutil.BaseSuite
	w *matrixWorld
}

func TestPermissionMatrixSuite(t *testing.T) {
	s := &PermissionMatrixTestSuite{}
	s.SuiteName = "Permission matrix"
	suite.Run(t, s)
}

// roles are the callers' names, in the order the matrix lists them.
var roles = []string{"admin", "headA", "staffA", "guestA", "indivA", "indiv", "headB", "visitor"}

// matrixWorld is the world every case runs in.
type matrixWorld struct {
	by                    map[string]*testutil.UserClient
	orgA, orgB            string
	wsA, wsB              string
	guestB                *testutil.UserClient
	gIndivPriv, gIndivPub string // the individual's private and public games
	gGuestA               string // the guest's game in workshop A
	gStaffA               string // staff A's game outside any workshop: organisation A's through its designer

	indivKey        string             // the individual's key (its own share)
	sIndiv, sGuestA string             // the individual's session, the guest's in workshop A
	link            testutil.GameShare // the individual's share link of their public game
	sLink           string             // the session of the guest that link made
	slugB           string             // workshop B's public page
	gPageB, gHidden string             // a public and a private game of workshop B
}

func (s *PermissionMatrixTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	w := &matrixWorld{by: map[string]*testutil.UserClient{}}
	admin := s.DevUser()
	member := func(org, role, name string) *testutil.UserClient {
		u := s.CreateUser(name)
		Must(u.AcceptInvite(Must(admin.InviteToInstitution(org, role, u.ID)).Id.String()))
		return u
	}
	w.orgA = Must(admin.CreateInstitution("Matrix A")).Id.String()
	w.orgB = Must(admin.CreateInstitution("Matrix B")).Id.String()
	headA, staffA, headB := member(w.orgA, "head", "m-head-a"), member(w.orgA, "staff", "m-staff-a"), member(w.orgB, "head", "m-head-b")
	w.wsA = Must(headA.CreateWorkshop(w.orgA, "Matrix Werkstatt A")).Id.String()
	w.wsB = Must(headB.CreateWorkshop(w.orgB, "Matrix Werkstatt B")).Id.String()
	guestA := Must(s.JoinWorkshop(*Must(headA.CreateWorkshopInvite(w.wsA)).InviteToken))
	w.guestB = Must(s.JoinWorkshop(*Must(headB.CreateWorkshopInvite(w.wsB)).InviteToken))
	indivA := s.CreateUser("m-indiv-a")
	MustSucceed(indivA.AcceptWorkshopInviteByToken(*Must(headA.CreateWorkshopInvite(w.wsA)).InviteToken))
	indiv := s.CreateUser("m-indiv")

	w.gIndivPriv = Must(indiv.UploadGame("alien-first-contact")).Id.String()
	w.gIndivPub = Must(indiv.UploadGame("alien-first-contact")).Id.String()
	Must(indiv.SetGamePublic(w.gIndivPub, true))
	w.gGuestA = Must(guestA.UploadGame("alien-first-contact")).Id.String()
	w.gStaffA = Must(staffA.UploadGame("alien-first-contact")).Id.String()

	// keys and play: the individual's own key, workshop A's from its head
	w.indivKey = Must(indiv.AddApiKey("mock-matrix-indiv", "Matrix Key", "mock")).Id.String()
	keyA := Must(headA.AddApiKey("mock-matrix-a", "Matrix A Key", "mock")).Id.String()
	shareA := Must(headA.ShareApiKeyWithInstitution(keyA, w.orgA)).Id.String()
	Must(headA.SetWorkshopApiKey(w.wsA, &shareA))
	w.sIndiv = Must(indiv.CreateSession(testutil.ForGame(w.gIndivPriv)))
	w.sGuestA = Must(guestA.CreateSession(testutil.ForGame(w.gGuestA)))
	w.link = Must(indiv.CreateGameShare(w.gIndivPub, w.indivKey, nil))
	w.sLink = Must(s.Public().CreateSession(testutil.ForShare(w.link.Token)))

	// workshop B has a public page, with a public game and a private one
	ws := Must(headB.EditWorkshop(w.wsB, func(u *testutil.WorkshopUpdate) { u.Public = true }))
	w.slugB = *ws.PublicSlug
	w.gPageB = Must(w.guestB.UploadGame("alien-first-contact")).Id.String()
	Must(w.guestB.SetGamePublic(w.gPageB, true))
	w.gHidden = Must(w.guestB.UploadGame("alien-first-contact")).Id.String()

	w.by = map[string]*testutil.UserClient{
		"admin": admin, "headA": headA, "staffA": staffA, "guestA": guestA, "indivA": indivA,
		"indiv": indiv, "headB": headB, "visitor": s.Public().AsUser(),
	}
	s.w = w
}

// cell is one case of the matrix: an action and the callers granted it;
// everyone else is denied access (401, 403, 404, or a list showing
// nothing).
type cell struct {
	action  string
	granted []string
	act     func(c *testutil.UserClient) error
}

// check runs every cell for every caller and reports each cell that differs
// from the table, together, so one run shows the whole gap.
func (s *PermissionMatrixTestSuite) check(cells []cell) {
	var wrong []string
	for _, c := range cells {
		granted := map[string]bool{}
		for _, r := range c.granted {
			granted[r] = true
		}
		for _, role := range roles {
			err := c.act(s.w.by[role])
			switch {
			case granted[role] && err != nil:
				wrong = append(wrong, fmt.Sprintf("%-40s %-8s refused, want granted: %v", c.action, role, err))
			case !granted[role] && err == nil:
				wrong = append(wrong, fmt.Sprintf("%-40s %-8s granted, want refused", c.action, role))
			case !granted[role] && !testutil.DeniedAccess(err):
				// a refusal is a denial of access, never a server error
				wrong = append(wrong, fmt.Sprintf("%-40s %-8s failed, want denied: %v", c.action, role, err))
			}
		}
	}
	sort.Strings(wrong)
	if len(wrong) > 0 {
		s.T().Errorf("%d cells differ from the matrix:\n%s", len(wrong), strings.Join(wrong, "\n"))
	}
}

// fresh runs a destructive action on a resource made anew for each caller.
func fresh(make func() string, act func(c *testutil.UserClient, id string) error) func(*testutil.UserClient) error {
	return func(c *testutil.UserClient) error { return act(c, make()) }
}

func (s *PermissionMatrixTestSuite) TestGames() {
	w := s.w
	indivGame := func() string { return Must(w.by["indiv"].UploadGame("alien-first-contact")).Id.String() }
	guestGame := func() string { return Must(w.by["guestA"].UploadGame("alien-first-contact")).Id.String() }
	staffGame := func() string { return Must(w.by["staffA"].UploadGame("alien-first-contact")).Id.String() }
	read := func(id string) func(*testutil.UserClient) error {
		return func(c *testutil.UserClient) error { _, err := c.GetGameByID(id); return err }
	}
	content := func(id string) func(*testutil.UserClient) error {
		return func(c *testutil.UserClient) error { _, err := c.GetGameYAML(id); return err }
	}
	rename := func(id string) func(*testutil.UserClient) error {
		return func(c *testutil.UserClient) error {
			_, err := c.RenameGame(id, testutil.Unique("Umbenannt"))
			return err
		}
	}
	copyGame := func(id string) func(*testutil.UserClient) error {
		return func(c *testutil.UserClient) error { _, err := c.CloneGame(id); return err }
	}
	del := func(c *testutil.UserClient, id string) error { return c.DeleteGame(id) }
	publish := func(c *testutil.UserClient, id string) error { _, err := c.SetGamePublic(id, true); return err }

	signedIn := []string{"admin", "headA", "staffA", "guestA", "indivA", "indiv", "headB"}
	s.check([]cell{
		// someone's private game: theirs and the admin's. Copying is reading
		// the content and creating a game
		{"read an individual's private game", []string{"admin", "indiv"}, read(w.gIndivPriv)},
		{"read its content", []string{"admin", "indiv"}, content(w.gIndivPriv)},
		{"rename it", []string{"admin", "indiv"}, rename(w.gIndivPriv)},
		{"copy it", []string{"admin", "indiv"}, copyGame(w.gIndivPriv)},
		{"delete it", []string{"admin", "indiv"}, fresh(indivGame, del)},
		{"publish it", []string{"admin", "indiv"}, fresh(indivGame, publish)},

		// a public game: everyone signed in reads and copies it. A visitor
		// reads public games through a workshop's public page only: the game
		// endpoints take a sign-in, a gate beyond {Read, GameCard, Public}
		{"read a public game", signedIn, read(w.gIndivPub)},
		{"read its content", signedIn, content(w.gIndivPub)},
		{"copy it", signedIn, copyGame(w.gIndivPub)},
		{"rename it", []string{"admin", "indiv"}, rename(w.gIndivPub)},

		// a guest's game in workshop A: the guest's, the workshop's, its
		// organisation's
		{"read a guest's game in workshop A", []string{"admin", "headA", "staffA", "guestA", "indivA"}, read(w.gGuestA)},
		{"read its content", []string{"admin", "headA", "staffA", "guestA", "indivA"}, content(w.gGuestA)},
		{"rename it", []string{"admin", "headA", "staffA", "guestA"}, rename(w.gGuestA)},
		{"copy it", []string{"admin", "headA", "staffA", "guestA", "indivA"}, copyGame(w.gGuestA)},
		{"delete it", []string{"admin", "headA", "staffA", "guestA"}, fresh(guestGame, del)},
		{"publish it", []string{"admin", "headA", "staffA", "guestA"}, fresh(guestGame, publish)},

		// staff A's game outside any workshop: organisation A's, through its
		// designer; head and staff work on it, only its designer deletes it
		{"read staff A's game outside a workshop", []string{"admin", "headA", "staffA"}, read(w.gStaffA)},
		{"read its content", []string{"admin", "headA", "staffA"}, content(w.gStaffA)},
		{"rename it", []string{"admin", "headA", "staffA"}, rename(w.gStaffA)},
		{"copy it", []string{"admin", "headA", "staffA"}, copyGame(w.gStaffA)},
		{"delete it", []string{"admin", "staffA"}, fresh(staffGame, del)},
		{"publish it", []string{"admin", "headA", "staffA"}, fresh(staffGame, publish)},
	})
}

func (s *PermissionMatrixTestSuite) TestWorkshops() {
	w := s.w
	workshopA := func() string {
		return Must(w.by["headA"].CreateWorkshop(w.orgA, testutil.Unique("Wegwerf"))).Id.String()
	}
	s.check([]cell{
		{"read workshop A", []string{"admin", "headA", "staffA", "guestA", "indivA"},
			func(c *testutil.UserClient) error { _, err := c.GetWorkshop(w.wsA); return err }},
		{"edit it", []string{"admin", "headA", "staffA"},
			func(c *testutil.UserClient) error {
				_, err := c.EditWorkshop(w.wsA, func(u *testutil.WorkshopUpdate) { u.Name = testutil.Unique("Werkstatt") })
				return err
			}},
		{"invite into it", []string{"admin", "headA", "staffA"},
			func(c *testutil.UserClient) error { _, err := c.CreateWorkshopInvite(w.wsA); return err }},
		{"delete a workshop of A", []string{"admin", "headA", "staffA"},
			fresh(workshopA, func(c *testutil.UserClient, id string) error { return c.DeleteWorkshop(id) })},
		{"create a workshop in A", []string{"admin", "headA", "staffA"},
			func(c *testutil.UserClient) error {
				_, err := c.CreateWorkshop(w.orgA, testutil.Unique("Neu"))
				return err
			}},
	})
}

func (s *PermissionMatrixTestSuite) TestOrganisations() {
	w := s.w
	s.check([]cell{
		// its card: its members, whoever takes part in its workshops, the admin
		{"read organisation A", []string{"admin", "headA", "staffA", "guestA", "indivA"},
			func(c *testutil.UserClient) error { _, err := c.GetInstitution(w.orgA); return err }},
		{"list its members", []string{"admin", "headA", "staffA"},
			func(c *testutil.UserClient) error {
				return testutil.NothingListed(c.GetInstitutionMembers(w.orgA))
			}},
		{"rename it", []string{"admin", "headA", "staffA"},
			func(c *testutil.UserClient) error {
				_, err := c.RenameInstitution(w.orgA, testutil.Unique("Organisation"))
				return err
			}},
	})
}

func (s *PermissionMatrixTestSuite) TestUsers() {
	w := s.w
	guestA, staffA := w.by["guestA"].ID, w.by["staffA"].ID
	newGuestA := func() string {
		return Must(s.JoinWorkshop(*Must(w.by["headA"].CreateWorkshopInvite(w.wsA)).InviteToken)).ID
	}
	s.check([]cell{
		// a guest of A: theirs, their organisation's head and staff, the admin
		{"read guest A", []string{"admin", "headA", "staffA", "guestA"},
			func(c *testutil.UserClient) error { _, err := c.GetUser(guestA); return err }},
		{"rename guest A", []string{"admin", "headA", "staffA", "guestA"},
			func(c *testutil.UserClient) error {
				_, err := c.UpdateUserName(guestA, testutil.Unique("Wolke"))
				return err
			}},
		{"read guest A's login code", []string{"headA", "staffA", "guestA"},
			func(c *testutil.UserClient) error { _, err := c.GetUserSecrets(guestA); return err }},
		{"delete a guest of A", []string{"admin", "headA", "staffA"},
			fresh(newGuestA, func(c *testutil.UserClient, id string) error { return c.DeleteUser(id) })},

		// a member of A: theirs, their organisation's, the admin
		{"read staff A", []string{"admin", "headA", "staffA"},
			func(c *testutil.UserClient) error { _, err := c.GetUser(staffA); return err }},
		{"rename staff A", []string{"admin", "staffA"},
			func(c *testutil.UserClient) error {
				_, err := c.UpdateUserName(staffA, testutil.Unique("Sonne"))
				return err
			}},
	})
}

func (s *PermissionMatrixTestSuite) TestSessions() {
	w := s.w
	play := func(id string) func(*testutil.UserClient) error {
		return func(c *testutil.UserClient) error { _, err := c.SessionState(id); return err }
	}
	inspect := func(id string) func(*testutil.UserClient) error {
		return func(c *testutil.UserClient) error { _, err := c.SessionTopology(id); return err }
	}
	indivSession := func() string { return Must(w.by["indiv"].CreateSession(testutil.ForGame(w.gIndivPriv))) }
	lists := func(id string) func(*testutil.UserClient) error {
		return func(c *testutil.UserClient) error {
			sessions, err := c.ListSessions()
			for _, ses := range sessions {
				if ses.Id.String() == id {
					return err
				}
			}
			return testutil.NothingListed([]testutil.UserSession{}, err)
		}
	}
	s.check([]cell{
		// an account's session is its own: nobody else plays, looks into,
		// lists or deletes it, the admin neither
		{"play an individual's session", []string{"indiv"}, play(w.sIndiv)},
		{"look into it", []string{"indiv"}, inspect(w.sIndiv)},
		{"list it", []string{"indiv"}, lists(w.sIndiv)},
		{"delete an individual's session", []string{"indiv"},
			fresh(indivSession, func(c *testutil.UserClient, id string) error { return c.DeleteSession(id) })},
		{"play a guest's session in workshop A", []string{"guestA"}, play(w.sGuestA)},

		// a share-link guest's session is played by its id: whoever holds it
		{"play a share-link guest's session", roles, play(w.sLink)},
		{"look into it: playing includes the insights", roles, inspect(w.sLink)},
	})
}

func (s *PermissionMatrixTestSuite) TestShareLinksAndKeys() {
	w := s.w
	newLink := func() string { return Must(w.by["indiv"].CreateGameShare(w.gIndivPub, w.indivKey, nil)).Id.String() }
	newKey := func() string {
		return Must(w.by["indiv"].AddApiKey("mock-"+testutil.Unique("k"), testutil.Unique("Key"), "mock")).Id.String()
	}
	s.check([]cell{
		// a share link is its maker's, its game's owner's and its key's owner's
		{"stop the individual's share link", []string{"indiv"},
			fresh(newLink, func(c *testutil.UserClient, id string) error { return c.DeleteGameShare(w.gIndivPub, id) })},

		// a key is its owner's alone
		{"read the individual's key", []string{"indiv"},
			func(c *testutil.UserClient) error { _, err := c.GetApiKeyInfo(w.indivKey); return err }},
		{"rename it", []string{"indiv"},
			func(c *testutil.UserClient) error {
				_, err := c.RenameApiKey(w.indivKey, testutil.Unique("Key"))
				return err
			}},
		{"delete a key of the individual", []string{"indiv"},
			fresh(newKey, func(c *testutil.UserClient, id string) error { return c.DeleteApiKey(id, true) })},
	})
}

func (s *PermissionMatrixTestSuite) TestInvitesAndSite() {
	w := s.w
	newInvite := func() string {
		return Must(w.by["headA"].InviteToInstitution(w.orgA, "staff", s.CreateUser(testutil.Unique("eingeladen")).ID)).Id.String()
	}
	invite := func(role string) func(*testutil.UserClient) error {
		return func(c *testutil.UserClient) error {
			_, err := c.InviteToInstitution(w.orgA, role, s.CreateUser(testutil.Unique("neu")).ID)
			return err
		}
	}
	s.check([]cell{
		// an organisation invites through its head and staff; heads only by
		// its head (and the admin)
		{"invite staff into A", []string{"admin", "headA", "staffA"}, invite("staff")},
		{"invite a head into A", []string{"admin", "headA"}, invite("head")},
		{"revoke an invite of A", []string{"admin", "headA", "staffA"},
			fresh(newInvite, func(c *testutil.UserClient, id string) error { return c.RevokeInvite(id) })},

		// the site's settings are the admin's
		{"read the site's settings", []string{"admin"},
			func(c *testutil.UserClient) error { _, err := c.GetSystemSettings(); return err }},
		// so are its updates: the versions it may upgrade to, and the upgrade
		{"see the site's updates", []string{"admin"},
			func(c *testutil.UserClient) error { _, err := c.GetUpdates(); return err }},
		// how the site presents itself is everyone's to see, before signing
		// in too; the admin sets it
		{"read the site's branding", roles,
			func(c *testutil.UserClient) error { _, err := c.GetSite(); return err }},
		{"change the site's branding", []string{"admin"},
			func(c *testutil.UserClient) error {
				_, err := c.SetSiteBranding(map[string]string{"siteName": "ChatGameLab"})
				return err
			}},
	})
}

func (s *PermissionMatrixTestSuite) TestOrganisationKeysAndSponsoring() {
	w := s.w
	ownKey := func(c *testutil.UserClient) (string, error) {
		k, err := c.AddApiKey("mock-"+testutil.Unique("k"), testutil.Unique("Key"), "mock")
		return k.Id.String(), err
	}
	staffShareInA := func() string {
		key := Must(w.by["staffA"].AddApiKey("mock-"+testutil.Unique("k"), testutil.Unique("Key"), "mock")).Id.String()
		return Must(w.by["staffA"].ShareApiKeyWithInstitution(key, w.orgA)).Id.String()
	}
	s.check([]cell{
		// a member gives their own key to their organisation; nobody else does
		{"give one's own key to A", []string{"headA", "staffA"},
			func(c *testutil.UserClient) error {
				key, err := ownKey(c)
				if err != nil {
					return err
				}
				_, err = c.ShareApiKeyWithInstitution(key, w.orgA)
				return err
			}},
		// its owner takes it back, and the organisation's head removes it
		{"remove a staff member's key from A", []string{"headA", "staffA"},
			fresh(staffShareInA, func(c *testutil.UserClient, id string) error { return c.DeleteApiKey(id, false) })},
		// its free use is the organisation's settings: its head and staff,
		// and the admin
		{"make a key of A its free use", []string{"admin", "headA", "staffA"},
			fresh(staffShareInA, func(c *testutil.UserClient, id string) error {
				_, err := c.SetInstitutionFreeUseApiKey(w.orgA, &id)
				return err
			})},

		// sponsoring a game takes editing it ({Edit, GameCard}): its owner,
		// and the admin; a public game is not anyone's to sponsor
		{"sponsor the individual's public game with one's own key", []string{"admin", "indiv"},
			func(c *testutil.UserClient) error {
				key, err := ownKey(c)
				if err != nil {
					return err
				}
				_, err = c.SetGameSponsor(w.gIndivPub, key)
				return err
			}},
	})
}

func (s *PermissionMatrixTestSuite) TestParticipantsAndStanding() {
	w := s.w
	newGuestA := func() string {
		return Must(s.JoinWorkshop(*Must(w.by["headA"].CreateWorkshopInvite(w.wsA)).InviteToken)).ID
	}
	s.check([]cell{
		// workshop A's head and staff (and the admin) take people out of it
		{"remove a guest from workshop A", []string{"admin", "headA", "staffA"},
			fresh(newGuestA, func(c *testutil.UserClient, id string) error { return c.RemoveMemberFromWorkshop(w.wsA, id) })},
		// a guest's login code is for their organisation's head and staff
		{"give a guest of A a new login code", []string{"headA", "staffA"},
			fresh(newGuestA, func(c *testutil.UserClient, id string) error { _, err := c.ResetGuestToken(id); return err })},

		// an organisation's standing: only the admin suspends it
		{"suspend organisation B and lift it", []string{"admin"},
			func(c *testutil.UserClient) error {
				if _, err := c.SuspendInstitution(w.orgB, "Matrix"); err != nil {
					return err
				}
				_, err := c.LiftSuspension(w.orgB)
				return err
			}},
	})
}

// An organisation's verified page (#340): its head and staff list and
// register it; the admin lists every one and alone approves it.
func (s *PermissionMatrixTestSuite) TestVerifiedPages() {
	w := s.w
	head := w.by["headA"]
	list := func(c *testutil.UserClient) error { _, err := c.ListVerifiedPages(w.orgA); return err }
	// an organisation has one page: each try removes it again
	register := func(c *testutil.UserClient) error {
		page, err := c.CreateVerifiedPage(w.orgA, "https://example.org/play", "https://example.org/cgl-callback")
		if err != nil {
			return err
		}
		return head.DeleteVerifiedPage(page.Id.String())
	}
	orgAPage := func() string {
		if pages := Must(head.ListVerifiedPages(w.orgA)); len(pages) > 0 {
			return pages[0].Id.String()
		}
		return Must(head.CreateVerifiedPage(w.orgA, "https://example.org/play", "https://example.org/cgl-callback")).Id.String()
	}
	approve := func(c *testutil.UserClient, id string) error { _, err := c.ApproveVerifiedPage(id, true); return err }
	s.check([]cell{
		{"list organisation A's verified page", []string{"admin", "headA", "staffA"}, list},
		{"register it", []string{"headA", "staffA"}, register},
		{"approve it", []string{"admin"}, fresh(orgAPage, approve)},
	})
}

// A workshop's public page is the one place where someone not signed in
// reads public games, download included: the workshop published them. A
// game it does not show is not there for anyone.
func (s *PermissionMatrixTestSuite) TestPublicPage() {
	w := s.w
	s.check([]cell{
		{"open workshop B's public page", roles,
			func(c *testutil.UserClient) error { _, err := c.GetPublicWorkshopPage(w.slugB); return err }},
		{"download a public game on it", roles,
			func(c *testutil.UserClient) error {
				_, err := c.GetPublicWorkshopGameYAML(w.slugB, w.gPageB)
				return err
			}},
		{"download a private game of its workshop", nil,
			func(c *testutil.UserClient) error {
				_, err := c.GetPublicWorkshopGameYAML(w.slugB, w.gHidden)
				return err
			}},
	})
}
