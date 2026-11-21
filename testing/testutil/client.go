// package: testutil / the tests' callers
// type:    logic
// job:     the three kinds of caller a test speaks as (an account with its bearer token, a browser with only its session cookie, an anonymous visitor), each on the generated apiclient, and the plumbing that turns its answers into values or refusals
// limits:  no suite lifecycle (-> suite.go); the words tests use live in the topical files
package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"cgl/apiclient"
)

// caller is what every kind of caller shares: the suite it belongs to and
// its generated client, which carries its credentials on every request.
type caller struct {
	s   *BaseSuite
	api *apiclient.ClientWithResponses
}

// t is the running test, so a failure lands in the test that caused it.
func (c caller) t() *testing.T { return c.s.T() }

// UserClient is an account signed in with its bearer token, as the web app
// and headless clients call.
type UserClient struct {
	caller
	Name  string
	ID    string
	Email string
	Token string
}

// PublicClient calls without any credentials, as a visitor or a guest does.
type PublicClient struct {
	caller
}

// CookieClient carries only a session cookie, as the playback-engine in its
// iframe does.
type CookieClient struct {
	caller
	cookie *http.Cookie
}

// newCaller builds a client against the suite's instance; editors add the
// caller's credentials to every request.
func newCaller(s *BaseSuite, editors ...apiclient.RequestEditorFn) caller {
	s.T().Helper()
	opts := make([]apiclient.ClientOption, 0, len(editors))
	for _, e := range editors {
		opts = append(opts, apiclient.WithRequestEditorFn(e))
	}
	api, err := apiclient.NewClientWithResponses(s.inst.URL(), opts...)
	if err != nil {
		s.T().Fatalf("building the API client: %v", err)
	}
	return caller{s: s, api: api}
}

// bearer signs every request with the token.
func bearer(s *BaseSuite, token string) apiclient.RequestEditorFn {
	header := s.inst.AuthHeader(token)
	return func(_ context.Context, req *http.Request) error {
		for k, v := range header {
			req.Header[k] = v
		}
		return nil
	}
}

// withCookie sends the cookie on every request.
func withCookie(cookie *http.Cookie) apiclient.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		req.AddCookie(cookie)
		return nil
	}
}

// header sets a request header.
func header(key, value string) apiclient.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		req.Header.Set(key, value)
		return nil
	}
}

// sameOrigin sends the Origin header a browser sends from the platform's own
// pages; schemaf refuses a cookie-signed request that changes state without it.
func sameOrigin(s *BaseSuite) apiclient.RequestEditorFn {
	return header("Origin", s.inst.URL())
}

// query adds a query parameter; the OpenAPI spec leaves query parameters
// out (schemaf#58), so the generated methods have no argument for them.
func query(key, value string) apiclient.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		q := req.URL.Query()
		q.Set(key, value)
		req.URL.RawQuery = q.Encode()
		return nil
	}
}

// ctx is the context of every call a test makes.
func ctx() context.Context { return context.Background() }

// Refusal is the API saying no: the status, and the code and message of its
// error body ({"error": …, "code": …}).
type Refusal struct {
	Status  int
	Code    string
	Message string
	Body    []byte
}

func (r *Refusal) Error() string {
	return fmt.Sprintf("refused with %d %s: %s", r.Status, r.Code, r.Message)
}

// response is what every generated response offers.
type response interface {
	StatusCode() int
	GetBody() []byte
}

// refused is the refusal a non-2xx answer carries, or nil.
func refused(r response) error {
	status := r.StatusCode()
	if status >= 200 && status < 300 {
		return nil
	}
	ref := &Refusal{Status: status, Body: r.GetBody()}
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.Unmarshal(ref.Body, &body); err == nil {
		ref.Code, ref.Message = body.Code, body.Error
	} else {
		ref.Message = string(ref.Body)
	}
	return ref
}

// some is a pointer to v, for the optional parameters of the generated client.
func some[T any](v T) *T { return &v }

// decode reads a successful answer's body as T, or returns the refusal.
func decode[T any](r response, err error) (T, error) {
	var out T
	if err != nil {
		return out, err
	}
	if err := refused(r); err != nil {
		return out, err
	}
	if body := r.GetBody(); len(body) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			return out, fmt.Errorf("reading the answer as %T: %w (body: %s)", out, err, body)
		}
	}
	return out, nil
}

// done reports whether the call succeeded, ignoring any body.
func done(r response, err error) error {
	if err != nil {
		return err
	}
	return refused(r)
}
