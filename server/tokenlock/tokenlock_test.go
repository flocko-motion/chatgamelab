package tokenlock

import (
	"testing"
	"time"
)

func newTestLock(max int) (*Lock, *time.Time) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	l := New(max, 10*time.Minute, 5*time.Minute)
	l.now = func() time.Time { return now }
	return l, &now
}

func TestLocksAfterMaxFailures(t *testing.T) {
	l, _ := newTestLock(3)
	for range 3 {
		l.Fail()
	}
	if l.RetryAfter() != 0 {
		t.Fatal("locked at max failures; must lock only above")
	}
	l.Fail()
	if got := l.RetryAfter(); got != 5*time.Minute {
		t.Fatalf("RetryAfter = %v, want 5m", got)
	}
}

func TestUnlocksAfterDuration(t *testing.T) {
	l, now := newTestLock(1)
	l.Fail()
	l.Fail()
	*now = now.Add(5*time.Minute + time.Second)
	if l.RetryAfter() != 0 {
		t.Fatal("still locked after duration")
	}
	// The counter restarts: a single failure must not relock.
	l.Fail()
	if l.RetryAfter() != 0 {
		t.Fatal("relocked by the first failure after a lock")
	}
}

func TestFailuresOutsideWindowExpire(t *testing.T) {
	l, now := newTestLock(2)
	l.Fail()
	l.Fail()
	*now = now.Add(11 * time.Minute)
	l.Fail()
	if l.RetryAfter() != 0 {
		t.Fatal("failures older than the window still counted")
	}
}
