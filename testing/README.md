# Integration tests

Go integration tests for the ChatGameLab platform: they drive the API as its
callers do and check what it answers.

## Running

```bash
cd testing
go test ./...                      # every suite
go test -run TestSessionSuite      # one suite
go test -run TestSessionSuite/TestSessionCookie
```

All they need is a running container runtime (Docker or Podman).

## How a suite runs

Each suite embeds `testutil.BaseSuite`. Its `SetupSuite` starts a fresh
instance of the platform through schemaf's test harness
(`schemaftest.Start(t, app.Wire)`): its own database, cloned from a template
with every migration applied, and the backend in-process on a free port, in
development mode. The first suite in a run starts one Postgres container; it
goes away with the test process.

The tests of a suite share its instance and database, so a test that depends
on another one shows up. `s.RestartBackend()` restarts the backend on the
same database, as a deployment does.

## Writing a test

A test reads like plain English, in the words `testutil` offers:

```go
type MyTestSuite struct {
    testutil.BaseSuite
}

func TestMySuite(t *testing.T) {
    suite.Run(t, &MyTestSuite{})
}

func (s *MyTestSuite) TestAnOutsiderCannotOpenTheWorkshop() {
    head := s.CreateUser("head")
    outsider := s.CreateUser("outsider")
    inst := Must(s.DevUser().CreateInstitution("Org"))
    Must(head.AcceptInvite(Must(s.DevUser().InviteToInstitution(inst.Id.String(), "head", head.ID)).Id.String()))
    workshop := Must(head.CreateWorkshop(inst.Id.String(), "Workshop"))

    _, err := outsider.GetWorkshop(workshop.Id.String())
    s.Forbidden(err)
}
```

- **Callers.** `s.CreateUser(name)` creates an account and signs it in with a
  token the instance mints; `s.DevUser()` is the preseeded admin;
  `s.Public()` calls without credentials; `s.JoinWorkshop(inviteToken)` and
  `s.ParticipantLogin(token)` sign participants in; `s.JoinWorkshopWithCookie`
  gives a caller carrying only the session cookie, as the playback-engine in
  its iframe does.
- **Answers** are the generated client's types, re-exported from `testutil`
  (`testutil.Game`, `testutil.Workshop`, …).
- **Refusals** are asserted by their code: `s.Forbidden(err)`,
  `s.NotFound(err)`, `s.Unauthorized(err)`, `s.Refused(err, testutil.CodeNoKey)`.
- **Success**: `Must(value, err)` returns the value or fails the test;
  `MustSucceed(err)` for calls without one.

Test files import only the vocabulary, testify, `uuid` and harmless standard
library; `vocabulary_test.go` checks it. A missing word goes into `testutil`,
built on the generated client `cgl/apiclient` (`docs/rules/test-vocabulary.md`).

## Layout

```
testing/
├── testutil/
│   ├── suite.go       # BaseSuite: the instance, creating and signing in callers
│   ├── client.go      # the callers on the generated client; refusals
│   ├── refusals.go    # error codes and the words for refusals
│   ├── types.go       # the generated client's types tests name
│   └── users.go, orgs.go, apikeys.go, games.go, sessions.go   # the words
├── testdata/games/    # game files UploadGame reads
├── vocabulary_test.go # test files speak only the vocabulary
└── *_test.go          # the suites
```
