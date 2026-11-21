package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"cgl/obj"

	engineapi "engine/api"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// registered is the platform's routes, as the generated Provider registers them.
func registered(t *testing.T) []schemafapi.Route {
	t.Helper()
	schemafapi.Reset()
	t.Cleanup(schemafapi.Reset)
	Provider()
	return schemafapi.Routes()
}

// Codegen only sees endpoints declared here, so the platform declares one per
// engine route by hand: every engine route has a platform endpoint with the
// same method and path, and no platform endpoint under /api/engine/ lacks one.
func TestEveryEngineRouteHasAPlatformEndpoint(t *testing.T) {
	var platform []string
	for _, r := range registered(t) {
		if strings.HasPrefix(r.Path, "/api/engine/") {
			platform = append(platform, r.Method+" "+r.Path)
		}
	}
	var engine []string
	for _, r := range engineapi.Routes() {
		engine = append(engine, r.Method+" "+r.Path)
	}
	for _, route := range engine {
		if !slices.Contains(platform, route) {
			t.Errorf("engine route %s has no platform endpoint", route)
		}
	}
	for _, route := range platform {
		if !slices.Contains(engine, route) {
			t.Errorf("platform endpoint %s has no engine route", route)
		}
	}
}

// A session the platform cannot hold is refused in schemaf's error format,
// with the engine's problem as the code, as the engine itself answers.
func TestAnUnknownSessionAnswersErrorAndCode(t *testing.T) {
	var state http.Handler
	for _, r := range registered(t) {
		if r.Method == "GET" && r.Path == "/api/engine/sessions/{id}/state" {
			state = r.Handler
		}
	}
	if state == nil {
		t.Fatal("no platform endpoint for a session's state")
	}
	mux := http.NewServeMux()
	mux.Handle("GET /api/engine/sessions/{id}/state", state)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/engine/sessions/not-a-session/state", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status %d, want 404", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"error": "there is no session not-a-session", "code": "unknown"}
	if fmt.Sprint(body) != fmt.Sprint(want) {
		t.Errorf("body = %v, want %v", body, want)
	}
}

// Only staff and heads change how often a session repeats the
// youth-protection rule.
func TestOnlyStaffAndHeadsTuneYouthProtection(t *testing.T) {
	for role, want := range map[obj.Role]bool{
		obj.RoleStaff: true, obj.RoleHead: true,
		obj.RoleAdmin: false, obj.RoleIndividual: false, obj.RoleParticipant: false,
	} {
		player := &obj.User{Role: &obj.UserRole{Role: role}}
		if got := tunesYouthProtection(player); got != want {
			t.Errorf("%s may tune: %v, want %v", role, got, want)
		}
	}
	if tunesYouthProtection(nil) || tunesYouthProtection(&obj.User{}) {
		t.Error("a guest or a player without a role may tune")
	}
}
