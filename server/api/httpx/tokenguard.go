// package: httpx / token lock middleware
// type:    wiring
// job:     answers 429 on routes that check guessable tokens while the site-wide token lock holds
// limits:  failure counting lives at the lookups (-> tokenlock, db)
package httpx

import (
	"cgl/tokenlock"
	"math"
	"net/http"
	"strconv"
	"time"
)

// ErrCodeTokenLocked tells the frontend to show "try again later".
const ErrCodeTokenLocked = "token_locked"

// TokenGuard rejects requests while the token lock holds.
func TokenGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if retry := tokenlock.RetryAfter(); retry > 0 {
			WriteTokenLocked(w, retry)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WriteTokenLocked writes 429 with Retry-After in whole seconds.
func WriteTokenLocked(w http.ResponseWriter, retry time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retry.Seconds()))))
	WriteErrorWithCode(w, http.StatusTooManyRequests, ErrCodeTokenLocked, "Too many failed attempts, try again later")
}
