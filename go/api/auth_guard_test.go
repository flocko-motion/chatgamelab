package api

import (
	"sort"
	"testing"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// openEndpoints are the endpoints a caller reaches without signing in: none
// (AuthNone) or optional (AuthOptional, anonymous allowed). Every other
// endpoint requires a sign-in. Opening an endpoint is a decision: it changes
// this list in the same review.
var openEndpoints = map[string]schemafapi.AuthLevel{
	// signing in and registering
	"GET /api/auth/check-name":   schemafapi.AuthNone,
	"POST /api/auth/dev-login":   schemafapi.AuthNone, // dev mode only
	"POST /api/auth/guest-login": schemafapi.AuthNone,
	"POST /api/users/new":        schemafapi.AuthNone, // dev mode only
	"GET /api/users/me":          schemafapi.AuthOptional,
	"GET /api/dev/mails":         schemafapi.AuthNone, // dev mode only

	// the declaration an organisation consents to, read before signing up
	"GET /api/legal/institution-declaration": schemafapi.AuthNone,

	// invites: an anonymous caller joins a workshop by token
	"GET /api/invites/{idOrToken}":         schemafapi.AuthOptional,
	"POST /api/invites/{idOrToken}/accept": schemafapi.AuthOptional,

	// public pages and guests playing through a share link
	"GET /api/play/{token}/info":                       schemafapi.AuthNone,
	"GET /api/public/workshops/{slug}":                 schemafapi.AuthNone,
	"GET /api/public/workshops/{slug}/games/{id}/yaml": schemafapi.AuthNone,
	"GET /api/public/workshops/{slug}/games/{id}/cgl":  schemafapi.AuthNone,

	// the engine: guests play too
	"POST /api/engine/sessions":                              schemafapi.AuthOptional,
	"POST /api/engine/sessions/{id}/choose":                  schemafapi.AuthOptional,
	"POST /api/engine/sessions/{id}/connect":                 schemafapi.AuthOptional,
	"POST /api/engine/sessions/{id}/input":                   schemafapi.AuthOptional,
	"POST /api/engine/sessions/{id}/pause":                   schemafapi.AuthOptional,
	"POST /api/engine/sessions/{id}/toggles":                 schemafapi.AuthOptional,
	"POST /api/engine/sessions/{id}/constraint-frequency":    schemafapi.AuthOptional,
	"POST /api/engine/sessions/{id}/nodes/{node}/compact":    schemafapi.AuthOptional,
	"POST /api/engine/sessions/{id}/nodes/{node}/inject":     schemafapi.AuthOptional,
	"POST /api/engine/sessions/{id}/nodes/{node}/beat":       schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/journal":                  schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/edges/{from}/{to}/{kind}": schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/flowchart":                schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/graph":                    schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/history":                  schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/live":                     schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/nodes/{node}":             schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/options":                  schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/prompts":                  schemafapi.AuthOptional,
	"PUT /api/engine/sessions/{id}/prompts/{name}":           schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/guardrail":                schemafapi.AuthOptional,
	"PUT /api/engine/sessions/{id}/guardrail":                schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/state":                    schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/stream":                   schemafapi.AuthOptional,
	"GET /api/engine/sessions/{id}/topology":                 schemafapi.AuthOptional,
	"GET /api/engine/playback-engine/":                       schemafapi.AuthOptional,
	"GET /api/engine/playback-engine/{file}":                 schemafapi.AuthOptional,
	"GET /api/engine/playback-engine/{dir}/{file}":           schemafapi.AuthOptional,

	// facts anyone may read
	"GET /api/languages":        schemafapi.AuthNone,
	"GET /api/languages/{code}": schemafapi.AuthNone,
	"GET /api/version":          schemafapi.AuthNone,
}

// Every endpoint requires a sign-in unless openEndpoints names it, with its
// level; and every endpoint the list names exists with that level.
func TestOnlyListedEndpointsAreOpen(t *testing.T) {
	schemafapi.Reset()
	t.Cleanup(schemafapi.Reset)
	Provider()

	open := map[string]schemafapi.AuthLevel{}
	for _, r := range schemafapi.Routes() {
		if r.Auth != schemafapi.AuthRequired {
			open[r.Method+" "+r.Path] = r.Auth
		}
	}

	var unlisted, stale []string
	for route, level := range open {
		if want, ok := openEndpoints[route]; !ok || want != level {
			unlisted = append(unlisted, "\""+route+"\": schemafapi."+level.String()+",")
		}
	}
	for route, level := range openEndpoints {
		if open[route] != level {
			stale = append(stale, route)
		}
	}
	sort.Strings(unlisted)
	sort.Strings(stale)
	for _, r := range unlisted {
		t.Errorf("open but not listed in openEndpoints (or at another level): %s", r)
	}
	for _, r := range stale {
		t.Errorf("listed in openEndpoints but not open at that level: %s", r)
	}
}
