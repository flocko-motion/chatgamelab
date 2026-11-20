package api

import (
	"net/http"
	"testing"

	"cgl/api/apitest"
	"cgl/obj"
	"cgl/tokenlock"
)

// Must run before anything else in this binary touches tokenlock.Default.
func TestTokenLocked(t *testing.T) {
	t.Setenv("TOKEN_LOCK_MAX_FAILURES", "2")
	t.Setenv("TOKEN_LOCK_DURATION", "90s")

	tokenlock.Fail()
	tokenlock.Fail()
	if rec, _ := apitest.ServeError(t, tokenLocked); rec.Code != http.StatusOK {
		t.Fatalf("status %d at the threshold, want 200", rec.Code)
	}

	tokenlock.Fail()
	rec, body := apitest.ServeError(t, tokenLocked)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d above the threshold, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "90" {
		t.Errorf("Retry-After = %q, want 90", got)
	}
	if body["code"] != codeTokenLocked {
		t.Errorf("code = %q, want %q", body["code"], codeTokenLocked)
	}
}

func TestAppErrorKeepsCode(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"profane name", obj.ErrProfaneName("not allowed"), http.StatusBadRequest, obj.ErrCodeProfaneName},
		{"last head", &obj.AppError{Code: obj.ErrCodeLastHead, Message: "needs a head"}, http.StatusConflict, obj.ErrCodeLastHead},
		{"validation", obj.ErrValidation("bad"), http.StatusBadRequest, obj.ErrCodeValidation},
		{"not found keeps schemaf's code", obj.ErrNotFound("gone"), http.StatusNotFound, "not_found"},
		{"forbidden", obj.ErrForbidden("no"), http.StatusForbidden, "forbidden"},
		{"http error", obj.NewHTTPErrorWithCode(http.StatusNotFound, "share_gone", "gone"), http.StatusNotFound, "share_gone"},
		{"code without a status keeps it on 500", &obj.AppError{Code: obj.ErrCodeNoApiKey, Message: "no key"}, http.StatusInternalServerError, obj.ErrCodeNoApiKey},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec, body := apitest.ServeError(t, func() error { return appError(c.err) })
			if rec.Code != c.status {
				t.Errorf("status %d, want %d", rec.Code, c.status)
			}
			if body["code"] != c.code {
				t.Errorf("code %q, want %q", body["code"], c.code)
			}
		})
	}
}
