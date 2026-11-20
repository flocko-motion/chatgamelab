// package: apitest / test support for api
// type:    test support
// job:     serves requests through schemaf's routes, so tests see the status, code, headers and body a client gets
// limits:  lives outside go/api/*.go, where schemaf's codegen would register its endpoint
package apitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// errorEndpoint answers with the error its function returns, 200 without one.
type errorEndpoint struct{ err func() error }

func (errorEndpoint) Method() string             { return "GET" }
func (errorEndpoint) Path() string               { return "/test/error" }
func (errorEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (e errorEndpoint) HandleRaw(w http.ResponseWriter, r *http.Request) error {
	if err := e.err(); err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

// ServeError answers a request with err's result as schemaf writes it, and
// the decoded error body (empty on 200).
func ServeError(t *testing.T, err func() error) (*httptest.ResponseRecorder, map[string]string) {
	t.Helper()
	rec := httptest.NewRecorder()
	route := schemafapi.NewRawRoute(errorEndpoint{err: err}, "", "")
	route.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test/error", nil))
	body := map[string]string{}
	if rec.Code != http.StatusOK {
		if decodeErr := json.NewDecoder(rec.Body).Decode(&body); decodeErr != nil {
			t.Fatalf("error body: %v", decodeErr)
		}
	}
	return rec, body
}

// Serve answers req through route, mounted at its method and path as schemaf
// mounts it, so path values resolve.
func Serve(t *testing.T, route schemafapi.Route, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(route.Method+" "+route.Path, route.Handler)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
