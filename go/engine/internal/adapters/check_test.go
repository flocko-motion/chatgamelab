package adapters

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// A caller that read the body for the detail still has no credit told apart.
func TestStatusErrorReadsNoCreditFromTheDetail(t *testing.T) {
	resp := &http.Response{StatusCode: http.StatusTooManyRequests, Status: "429", Body: io.NopCloser(strings.NewReader(""))}
	err := StatusError("call", resp, `: {"error":{"code":"credit_balance_exhausted"}}`)
	if !errors.Is(err, ErrNoCredit) || errors.Is(err, ErrTransient) {
		t.Errorf("got %v, want no credit and nothing to retry", err)
	}
}

// An account with no credit left is told apart from a rate limit: both answer
// 429, but only the rate limit passes on a retry. 402 says the same as the
// quota code.
func TestStatusErrorTellsNoCreditFromARateLimit(t *testing.T) {
	for name, tc := range map[string]struct {
		status              int
		body                string
		noCredit, transient bool
	}{
		"quota used up":    {http.StatusTooManyRequests, `{"error":{"code":"insufficient_quota"}}`, true, false},
		"credit exhausted": {http.StatusTooManyRequests, `{"error":{"type":"insufficient_quota","code":"credit_balance_exhausted"}}`, true, false},
		"payment required": {http.StatusPaymentRequired, `{}`, true, false},
		"rate limit":       {http.StatusTooManyRequests, `{"error":{"code":"rate_limit_exceeded"}}`, false, true},
	} {
		t.Run(name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tc.status, Status: http.StatusText(tc.status), Body: io.NopCloser(strings.NewReader(tc.body))}
			err := StatusError("call", resp, "")
			if got := errors.Is(err, ErrNoCredit); got != tc.noCredit {
				t.Errorf("no credit: got %v, want %v (%v)", got, tc.noCredit, err)
			}
			if got := errors.Is(err, ErrTransient); got != tc.transient {
				t.Errorf("transient: got %v, want %v (%v)", got, tc.transient, err)
			}
		})
	}
}

// A block retries what may pass and gives up on what will not, so the status
// has to say which it is: a timeout, a rate limit or a server failure may pass.
func TestStatusErrorMarksTransientFailures(t *testing.T) {
	for status, want := range map[int]struct{ transient, rejected bool }{
		http.StatusRequestTimeout:      {transient: true},
		http.StatusTooManyRequests:     {transient: true},
		http.StatusServiceUnavailable:  {transient: true},
		http.StatusInternalServerError: {transient: true},
		http.StatusBadRequest:          {},
		http.StatusNotFound:            {},
		http.StatusUnauthorized:        {rejected: true},
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			resp := &http.Response{StatusCode: status, Status: http.StatusText(status)}
			err := StatusError("call", resp, "")
			if got := errors.Is(err, ErrTransient); got != want.transient {
				t.Errorf("transient: got %v, want %v (%v)", got, want.transient, err)
			}
			if got := errors.Is(err, ErrKeyRejected); got != want.rejected {
				t.Errorf("key rejected: got %v, want %v (%v)", got, want.rejected, err)
			}
		})
	}
}
