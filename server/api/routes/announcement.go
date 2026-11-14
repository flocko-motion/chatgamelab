// package: routes / dashboard announcement HTTP handler
// type:    logic
// job:     reports the dashboard announcement that is active right now, if any, so time-limited notices appear and vanish without a deployment
// limits:  no route registration (-> router.go); the announcement texts live in the web locales, the server only decides whether and when
package routes

import (
	"net/http"
	"time"

	"cgl/api/httpx"
	"cgl/functional"
)

// The announcement window, kept in one place. The defaults carry an explicit
// offset (+01:00 = CET, Europe/Berlin in winter) so they never depend on the
// server's local time zone. Both ends can be moved by environment variable
// (RFC 3339 with offset, e.g. 2026-11-13T22:30:00+01:00) without a code change.
const (
	awardAnnouncementID           = "dieter-baacke-2026"
	awardAnnouncementLink         = "https://jff.de/link/gewonnen"
	awardAnnouncementDefaultStart = "2026-11-13T22:30:00+01:00"
	awardAnnouncementDefaultEnd   = "2026-12-14T00:00:00+01:00"
	awardAnnouncementStartEnv     = "ANNOUNCEMENT_AWARD_START"
	awardAnnouncementEndEnv       = "ANNOUNCEMENT_AWARD_END"
)

// AnnouncementResponse tells the dashboard whether to show a notice. Outside the
// window only Active=false is sent, so nothing about the notice leaks early.
type AnnouncementResponse struct {
	Active   bool       `json:"active"`
	ID       string     `json:"id,omitempty"`
	Link     string     `json:"link,omitempty"`
	StartsAt *time.Time `json:"startsAt,omitempty"`
	EndsAt   *time.Time `json:"endsAt,omitempty"`
}

// GetCurrentAnnouncement godoc
//
//	@Summary		Get the current dashboard announcement
//	@Description	Returns the announcement that is active right now. Outside its time window only {"active": false} is returned.
//	@Tags			announcements
//	@Produce		json
//	@Success		200	{object}	AnnouncementResponse
//	@Router			/announcements/current [get]
func GetCurrentAnnouncement(w http.ResponseWriter, r *http.Request) {
	httpx.SetNoCacheHeaders(w)
	httpx.WriteJSON(w, http.StatusOK, currentAnnouncement(time.Now()))
}

// currentAnnouncement decides for a given instant, so tests can simulate the clock.
// The window is half-open: shown from start (inclusive) until end (exclusive).
func currentAnnouncement(now time.Time) AnnouncementResponse {
	start := announcementTime(awardAnnouncementStartEnv, awardAnnouncementDefaultStart)
	end := announcementTime(awardAnnouncementEndEnv, awardAnnouncementDefaultEnd)

	if now.Before(start) || !now.Before(end) {
		return AnnouncementResponse{Active: false}
	}
	return AnnouncementResponse{
		Active:   true,
		ID:       awardAnnouncementID,
		Link:     awardAnnouncementLink,
		StartsAt: &start,
		EndsAt:   &end,
	}
}

// announcementTime reads an RFC 3339 time from the environment and falls back
// to the default when the variable is unset or unparsable, so a typo can never
// switch the notice on early.
func announcementTime(envName, defaultValue string) time.Time {
	if t, err := time.Parse(time.RFC3339, functional.EnvOrDefault(envName, defaultValue)); err == nil {
		return t
	}
	t, _ := time.Parse(time.RFC3339, defaultValue)
	return t
}
