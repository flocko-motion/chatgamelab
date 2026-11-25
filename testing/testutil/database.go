// package: testutil / the words for what the database itself refuses
// type:    logic
// job:     writes straight to the suite's database, past the core, and checks what its constraints refuse
// limits:  for the guarantees that hold whoever writes; everything else goes through the API
package testutil

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/lib/pq"
)

// DatabaseRefuses asserts the database refuses the statement as breaking
// one of its rules (a check constraint or trigger: SQLSTATE 23514).
func (s *BaseSuite) DatabaseRefuses(query string, msgAndArgs ...any) bool {
	s.T().Helper()
	_, err := s.inst.DB().Exec(query)
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return s.Fail(fmt.Sprintf("expected the database to refuse %q, got %v", query, err), msgAndArgs...)
	}
	return s.Equal(pq.ErrorCode("23514"), pqErr.Code, append([]any{"refused for %s: %s", pqErr.Code.Name(), pqErr.Message}, msgAndArgs...)...)
}

// DatabaseAccepts asserts the database takes the statement.
func (s *BaseSuite) DatabaseAccepts(query string, msgAndArgs ...any) bool {
	s.T().Helper()
	_, err := s.inst.DB().Exec(query)
	return s.NoError(err, msgAndArgs...)
}

var unique atomic.Int64

// Unique is prefix with a number no other call gets: a name no case shares.
func Unique(prefix string) string { return fmt.Sprintf("%s %d", prefix, unique.Add(1)) }

// DatabaseCount is the count a counting query answers, past the core.
func (s *BaseSuite) DatabaseCount(query string) int {
	s.T().Helper()
	var n int
	if err := s.inst.DB().QueryRow(query).Scan(&n); err != nil {
		s.T().Fatalf("counting with %q: %v", query, err)
	}
	return n
}

// SessionIdleFor makes a session look not played for that many hours, past
// the core: the janitor goes by a session's last change.
func (s *BaseSuite) SessionIdleFor(sessionID string, hours int) {
	s.T().Helper()
	s.DatabaseAccepts(fmt.Sprintf("UPDATE game_session SET modified_at = now() - interval '%d hours' WHERE id = '%s'", hours, sessionID))
}

// SessionStored is whether the session is still in the database.
func (s *BaseSuite) SessionStored(sessionID string) bool {
	s.T().Helper()
	return s.DatabaseCount(fmt.Sprintf("SELECT count(*) FROM game_session WHERE id = '%s'", sessionID)) == 1
}

// AccountStored is whether the account is still in the database.
func (s *BaseSuite) AccountStored(userID string) bool {
	s.T().Helper()
	return s.DatabaseCount(fmt.Sprintf("SELECT count(*) FROM app_user WHERE id = '%s'", userID)) == 1
}

// SignUpStartedHoursAgo makes an account in sign-up look started that many
// hours ago, past the core.
func (s *BaseSuite) SignUpStartedHoursAgo(userID string, hours int) {
	s.T().Helper()
	s.DatabaseAccepts(fmt.Sprintf("UPDATE app_user SET created_at = now() - interval '%d hours' WHERE id = '%s'", hours, userID))
}
