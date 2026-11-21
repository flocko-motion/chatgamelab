// package: game / server-side API key resolution
// type:    logic
// job:     resolves which API key and AI quality tier a play uses by walking the priority chain (workshop, sponsor, institution, user, system).
// limits:  does not call the AI or start plays; only selects and traces candidate keys for callers to apply.
package game

import (
	"context"
	"fmt"
	"strings"

	"cgl/core"
	"cgl/log"
	"cgl/obj"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
)

// resolvedKey holds the resolved API key share and the AI quality tier from the same source.
type resolvedKey struct {
	Share         *obj.ApiKeyShare
	AiQualityTier string // resolved tier (high/medium/low), never empty
	KeyType       string // source type: obj.ApiKeyType* constants
}

const maxCandidates = 3

// keyTraceStep records the outcome of one source in the priority chain.
type keyTraceStep struct {
	Source   string // e.g. "workshop", "sponsor", ...
	Result   string // "selected", "skipped", "filtered", "error"
	Detail   string // human-readable reason
	KeyName  string // API key name (if resolved)
	Platform string // API key platform (if resolved)
	Tier     string // resolved quality tier (if selected)
}

// resolveApiKeyCandidates are the keys that may pay for the caller's
// session of game, in order: the secure core decides (core.Payers); with
// platformFilter, only that platform's (resuming). userID is the caller, as
// v1 callers pass it.
func resolveApiKeyCandidates(ctx context.Context, userID uuid.UUID, gameID uuid.UUID, platformFilter string) ([]resolvedKey, *obj.HTTPError) {
	payers, steps, err := core.Payers(ctx, gameID, platformFilter)
	if err != nil {
		return nil, obj.NewHTTPErrorWithCode(500, obj.ErrCodeServerError, "Failed to resolve an API key")
	}
	var trace []keyTraceStep
	for _, st := range steps {
		step := keyTraceStep{Source: st.Source, Result: st.Result, Detail: st.Detail}
		if st.Payer != nil {
			step.KeyName, step.Platform = st.Payer.KeyName, st.Payer.Platform
			if st.Result == "selected" {
				step.Tier = st.Payer.Tier
			}
		}
		trace = append(trace, step)
	}
	candidates := make([]resolvedKey, len(payers))
	for i, p := range payers {
		share := &obj.ApiKeyShare{
			ApiKeyID: p.KeyID,
			ApiKey:   &obj.ApiKey{ID: p.KeyID, Name: p.KeyName, Platform: p.Platform},
		}
		if p.ShareID != nil {
			share.ID = *p.ShareID
		}
		candidates[i] = resolvedKey{Share: share, AiQualityTier: p.Tier, KeyType: p.Source}
	}
	sendKeyResolutionTrace(userID, gameID, platformFilter, candidates, trace)
	if len(candidates) == 0 {
		if platformFilter != "" {
			log.Debug("no API key available for platform", "user_id", userID, "game_id", gameID, "platform", platformFilter)
			return nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeNoApiKey, "No API key available for platform "+platformFilter+". All available keys use a different AI platform.")
		}
		log.Debug("no API key available", "user_id", userID, "game_id", gameID)
		return nil, obj.NewHTTPErrorWithCode(400, obj.ErrCodeNoApiKey, "No API key available. Please configure an API key in your settings.")
	}
	return candidates, nil
}

func sendKeyResolutionTrace(userID uuid.UUID, gameID uuid.UUID, platformFilter string, candidates []resolvedKey, trace []keyTraceStep) {
	// Build a human-readable summary
	var lines []string
	for i, step := range trace {
		line := fmt.Sprintf("%d. [%s] %s", i+1, step.Source, step.Result)
		if step.KeyName != "" {
			line += fmt.Sprintf(" key=%q platform=%s", step.KeyName, step.Platform)
		}
		if step.Tier != "" {
			line += fmt.Sprintf(" tier=%s", step.Tier)
		}
		if step.Detail != "" {
			line += fmt.Sprintf(" (%s)", step.Detail)
		}
		lines = append(lines, line)
	}
	summary := strings.Join(lines, "\n")

	// Build structured extras
	extras := map[string]any{
		"user_id":         userID.String(),
		"game_id":         gameID.String(),
		"platform_filter": platformFilter,
		"candidates":      len(candidates),
		"trace":           summary,
	}
	for i, c := range candidates {
		extras[fmt.Sprintf("candidate_%d_type", i)] = c.KeyType
		extras[fmt.Sprintf("candidate_%d_key", i)] = c.Share.ApiKey.Name
		extras[fmt.Sprintf("candidate_%d_platform", i)] = c.Share.ApiKey.Platform
		extras[fmt.Sprintf("candidate_%d_tier", i)] = c.AiQualityTier
	}

	event := sentry.NewEvent()
	event.Level = sentry.LevelInfo
	event.Message = fmt.Sprintf("API key resolution: %d candidates", len(candidates))
	event.Tags = map[string]string{
		"user_id":    userID.String(),
		"game_id":    gameID.String(),
		"candidates": fmt.Sprintf("%d", len(candidates)),
	}
	if len(candidates) > 0 {
		event.Tags["primary_key_type"] = candidates[0].KeyType
		event.Tags["primary_platform"] = candidates[0].Share.ApiKey.Platform
	}
	if platformFilter != "" {
		event.Tags["platform_filter"] = platformFilter
	}
	event.Extra = extras

	sentry.CaptureEvent(event)
}

// resolveApiKey resolves the highest-priority API key on any platform.
func resolveApiKey(ctx context.Context, userID uuid.UUID, gameID uuid.UUID) (*resolvedKey, *obj.HTTPError) {
	candidates, httpErr := resolveApiKeyCandidates(ctx, userID, gameID, "")
	if httpErr != nil {
		return nil, httpErr
	}
	return &candidates[0], nil
}
