// package: session / engine sessions from the platform's data
// type:    logic
// job:     reports which keys may pay for a session, and why the others don't, to Sentry.
// limits:  the payers are the core's (core.Payers); this only traces them.
package session

import (
	"fmt"
	"strings"

	"cgl/core"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
)

// traceKeys sends the payer resolution of a session of game to Sentry: each
// source, whether it counted, and why.
func traceKeys(gameID uuid.UUID, platform string, payers []core.Payer, steps []core.PayerStep) {
	lines := make([]string, len(steps))
	for i, st := range steps {
		line := fmt.Sprintf("%d. [%s] %s", i+1, st.Source, st.Result)
		if st.Payer != nil {
			line += fmt.Sprintf(" key=%q platform=%s", st.Payer.KeyName, st.Payer.Platform)
			if st.Result == "selected" {
				line += " tier=" + st.Payer.Tier
			}
		}
		if st.Detail != "" {
			line += " (" + st.Detail + ")"
		}
		lines[i] = line
	}
	extras := map[string]any{
		"game_id":         gameID.String(),
		"platform_filter": platform,
		"candidates":      len(payers),
		"trace":           strings.Join(lines, "\n"),
	}
	for i, p := range payers {
		extras[fmt.Sprintf("candidate_%d_type", i)] = p.Source
		extras[fmt.Sprintf("candidate_%d_key", i)] = p.KeyName
		extras[fmt.Sprintf("candidate_%d_platform", i)] = p.Platform
		extras[fmt.Sprintf("candidate_%d_tier", i)] = p.Tier
	}
	event := sentry.NewEvent()
	event.Level = sentry.LevelInfo
	event.Message = fmt.Sprintf("API key resolution: %d candidates", len(payers))
	event.Tags = map[string]string{"game_id": gameID.String(), "candidates": fmt.Sprintf("%d", len(payers))}
	if len(payers) > 0 {
		event.Tags["primary_key_type"] = payers[0].Source
		event.Tags["primary_platform"] = payers[0].Platform
	}
	if platform != "" {
		event.Tags["platform_filter"] = platform
	}
	event.Extra = extras
	sentry.CaptureEvent(event)
}
