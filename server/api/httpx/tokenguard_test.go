package httpx

import (
	"cgl/tokenlock"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Must run before anything else in this binary touches tokenlock.Default.
func TestTokenGuard(t *testing.T) {
	t.Setenv("TOKEN_LOCK_MAX_FAILURES", "2")
	t.Setenv("TOKEN_LOCK_DURATION", "90s")

	guarded := TokenGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	serve := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		guarded.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		return rec
	}

	tokenlock.Fail()
	tokenlock.Fail()
	if rec := serve(); rec.Code != http.StatusOK {
		t.Fatalf("status %d at the threshold, want 200", rec.Code)
	}

	tokenlock.Fail()
	rec := serve()
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d above the threshold, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "90" {
		t.Errorf("Retry-After = %q, want 90", got)
	}
}
