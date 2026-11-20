package adapters

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func key(context.Context) (string, error) { return "sk-test", nil }

// The platform shows a dead key and a provider that is down differently, so
// the check must keep them apart.
func TestCheckKeyTellsRejectedFromUnreachable(t *testing.T) {
	answering := func(status int) string {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer sk-test" {
				t.Errorf("the key was not sent: %q", r.Header.Get("Authorization"))
			}
			w.WriteHeader(status)
		}))
		t.Cleanup(srv.Close)
		return srv.URL
	}
	gone := httptest.NewServer(http.NotFoundHandler())
	gone.Close()

	for name, tc := range map[string]struct {
		url  string
		want error
	}{
		"accepted":     {answering(http.StatusOK), nil},
		"unauthorized": {answering(http.StatusUnauthorized), ErrKeyRejected},
		"forbidden":    {answering(http.StatusForbidden), ErrKeyRejected},
		"failing":      {answering(http.StatusBadGateway), ErrUnreachable},
		"down":         {gone.URL, ErrUnreachable},
	} {
		t.Run(name, func(t *testing.T) {
			err := CheckKey(context.Background(), key, tc.url)
			if tc.want == nil && err != nil {
				t.Fatalf("got %v, want the key accepted", err)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
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
