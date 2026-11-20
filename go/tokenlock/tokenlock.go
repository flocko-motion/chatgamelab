// package: tokenlock / site-wide lock against guessing tokens
// type:    logic
// job:     counts failed lookups of guessable tokens and locks all token checks once too many fail
// limits:  in-memory, so one backend instance; HTTP responses live in httpx.TokenGuard
package tokenlock

import (
	"cgl/functional"
	"cgl/log"
	"strconv"
	"sync"
	"time"
)

// Lock counts failures in a sliding window. Past maxFailures it locks every
// token check for duration, correct tokens included: a lock that let correct
// tokens through would still answer guesses. The lock is what keeps 3- and
// 4-word tokens (33/44 bits) out of reach of brute force.
type Lock struct {
	mu          sync.Mutex
	maxFailures int
	window      time.Duration
	duration    time.Duration
	failures    []time.Time
	lockedUntil time.Time
	now         func() time.Time
}

func New(maxFailures int, window, duration time.Duration) *Lock {
	return &Lock{maxFailures: maxFailures, window: window, duration: duration, now: time.Now}
}

// RetryAfter returns how long the lock still holds; zero when unlocked.
func (l *Lock) RetryAfter() time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	if d := l.lockedUntil.Sub(l.now()); d > 0 {
		return d
	}
	return 0
}

// Fail records a lookup of a token that was never issued.
func (l *Lock) Fail() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.window)
	kept := l.failures[:0]
	for _, t := range l.failures {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.failures = append(kept, now)
	if len(l.failures) > l.maxFailures {
		l.lockedUntil = now.Add(l.duration)
		l.failures = l.failures[:0]
		log.Warn("token lock engaged: too many failed token lookups",
			"failures", l.maxFailures, "window", l.window, "duration", l.duration)
	}
}

var (
	defaultLock *Lock
	defaultOnce sync.Once
)

// Default is the process-wide lock, configured on first use from
// TOKEN_LOCK_MAX_FAILURES, TOKEN_LOCK_WINDOW and TOKEN_LOCK_DURATION.
func Default() *Lock {
	defaultOnce.Do(func() {
		defaultLock = New(
			envInt("TOKEN_LOCK_MAX_FAILURES", 300),
			envDuration("TOKEN_LOCK_WINDOW", 10*time.Minute),
			envDuration("TOKEN_LOCK_DURATION", 5*time.Minute),
		)
	})
	return defaultLock
}

// Fail records a failure on the default lock.
func Fail() { Default().Fail() }

// RetryAfter reads the default lock.
func RetryAfter() time.Duration { return Default().RetryAfter() }

func envInt(name string, fallback int) int {
	v, err := strconv.Atoi(functional.EnvOrDefault(name, strconv.Itoa(fallback)))
	if err != nil || v < 1 {
		log.Warn("invalid env, using default", "name", name, "default", fallback)
		return fallback
	}
	return v
}

func envDuration(name string, fallback time.Duration) time.Duration {
	v, err := time.ParseDuration(functional.EnvOrDefault(name, fallback.String()))
	if err != nil || v <= 0 {
		log.Warn("invalid env, using default", "name", name, "default", fallback)
		return fallback
	}
	return v
}
