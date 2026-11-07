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
