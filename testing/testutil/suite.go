// package: testutil / integration test suite harness
// type:    logic
// job:     starts a fresh instance of the platform per suite (schemaf's harness: its own database, the backend in-process) and signs in the people a test speaks as
// limits:  test infrastructure only; the words tests use live in the topical files (-> users.go, games.go, …)
package testutil

import (
	"os"
	"strings"

	"cgl/apiclient"
	"cgl/app"

	lorem "github.com/drhodes/golorem"
	"github.com/flocko-motion/schemaf/schemaftest"
	"github.com/stretchr/testify/suite"
)

// devAdminID is the admin the database is preseeded with in development mode.
const devAdminID = "00000000-0000-0000-0000-000000000001"

// BaseSuite gives a suite its own instance of the platform. Embed it in a
// suite; its tests share the instance and its database.
type BaseSuite struct {
	suite.Suite
	SuiteName string
	inst      *schemaftest.Instance
	devUser   *UserClient
}

// SetupSuite starts the suite's instance, in development mode, and signs in
// the preseeded dev admin. The preseed is done before the server takes
// requests, so the admin is there from the first one.
func (s *BaseSuite) SetupSuite() {
	s.inst = schemaftest.Start(s.T(), app.Wire)
	s.devUser = s.signedIn(devAdminID, "dev", "dev@chatgamelab.local")
	if _, err := s.devUser.GetMe(); err != nil {
		s.T().Fatalf("the preseeded dev admin isn't there at start: %v", err)
	}
}

// RestartBackend stops the backend and starts it again on the same port and
// database, as a deployment does: whatever it held in memory is gone.
func (s *BaseSuite) RestartBackend() {
	s.T().Helper()
	s.inst.Restart()
}

// signedIn is the account with that id, signed in with a token minted by the
// instance.
func (s *BaseSuite) signedIn(id, name, email string) *UserClient {
	token := s.inst.Token(id)
	return &UserClient{
		caller: newCaller(s, bearer(s, token)),
		Name:   name,
		ID:     id,
		Email:  email,
		Token:  token,
	}
}

// CreateUser creates an account and signs it in. Name and email are
// optional: a missing name is a random word, a missing email is
// <name>@test.local.
// Example: alice := s.CreateUser("alice")
func (s *BaseSuite) CreateUser(nameAndEmail ...string) *UserClient {
	s.T().Helper()
	name := strings.ToLower(lorem.Word(1, 2))
	if len(nameAndEmail) > 0 && nameAndEmail[0] != "" {
		name = nameAndEmail[0]
	}
	email := name + "@test.local"
	if len(nameAndEmail) > 1 && nameAndEmail[1] != "" {
		email = nameAndEmail[1]
	}

	created, err := decode[User](s.Public().api.PostApiUsersNewWithResponse(ctx(), apiclient.CreateUserReq{Name: name, Email: &email}))
	if err != nil {
		s.T().Fatalf("creating user %q: %v", name, err)
	}
	s.T().Logf("created user %q (%s)", name, created.Id)
	return s.signedIn(created.Id.String(), name, email)
}

// CreateRegisteringUser creates an account as a first Auth0 sign-in does,
// with no name and no role, and signs it in: it is still registering.
func (s *BaseSuite) CreateRegisteringUser() *UserClient {
	s.T().Helper()
	created, err := decode[User](s.Public().api.PostApiUsersNewWithResponse(ctx(), apiclient.CreateUserReq{}))
	if err != nil {
		s.T().Fatalf("creating a registering account: %v", err)
	}
	return s.signedIn(created.Id.String(), "", "")
}

// Role gives the user a role, as the dev admin.
func (s *BaseSuite) Role(user *UserClient, role string) {
	s.T().Helper()
	if _, err := s.devUser.SetUserRole(user.ID, role); err != nil {
		s.T().Fatalf("assigning role %q to %q: %v", role, user.Name, err)
	}
}

// Public is a caller without any credentials.
func (s *BaseSuite) Public() *PublicClient {
	return &PublicClient{caller: newCaller(s)}
}

// DevUser is the preseeded dev admin.
func (s *BaseSuite) DevUser() *UserClient {
	return s.devUser
}

// TokenLockTestRequested reports whether the token lock suite was asked for
// with a threshold of 20 failures:
//
//	TOKEN_LOCK_TEST=1 TOKEN_LOCK_MAX_FAILURES=20 TOKEN_LOCK_DURATION=3s go test -run TestTokenLockSuite
func TokenLockTestRequested() bool {
	return os.Getenv("TOKEN_LOCK_TEST") == "1" && os.Getenv("TOKEN_LOCK_MAX_FAILURES") == "20"
}

