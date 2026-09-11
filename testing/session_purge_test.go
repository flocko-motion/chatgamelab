package testing

import (
	"cgl/db"
	"cgl/testing/testutil"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// SessionPurgeTestSuite covers the retention purge that keeps idle game sessions
// from accumulating in the database.
type SessionPurgeTestSuite struct {
	testutil.BaseSuite
}

func TestSessionPurgeTestSuite(t *testing.T) {
	suite.Run(t, new(SessionPurgeTestSuite))
}

// TestPurgeRemovesIdleSessionsOnly verifies that a session survives a cutoff that
// predates it and is deleted, messages included, by a cutoff that postdates it.
func (s *SessionPurgeTestSuite) TestPurgeRemovesIdleSessionsOnly() {
	ctx := context.Background()

	user := s.CreateUser("sp-session-user")
	Must(user.AddApiKey("mock-sp-sess", "Session Key", "mock"))

	game := Must(user.UploadGame("alien-first-contact"))
	session, err := user.CreateGameSession(game.ID.String())
	s.Require().NoError(err)
	s.Require().NotEmpty(session.ID)
	s.T().Logf("Created session: %s", session.ID)

	purged, err := db.PurgeExpiredSessions(ctx, time.Now().Add(-time.Hour))
	s.NoError(err)
	s.Zero(purged, "a session inside the retention window must survive")

	_, err = user.GetGameSession(session.ID.String())
	s.NoError(err, "session should still be readable after the purge")

	purged, err = db.PurgeExpiredSessions(ctx, time.Now().Add(time.Minute))
	s.NoError(err)
	s.GreaterOrEqual(purged, int64(1), "an expired session must be purged")

	_, err = user.GetGameSession(session.ID.String())
	s.Error(err, "session should be gone after the purge")

	remaining, err := db.CountExpiredSessions(ctx, time.Now().Add(time.Minute))
	s.NoError(err)
	s.Zero(remaining, "purge should drain every expired session")
	s.T().Logf("Purged %d session(s)", purged)
}
