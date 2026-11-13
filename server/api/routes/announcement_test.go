package routes

import (
	"testing"
	"time"
	_ "time/tzdata" // Europe/Berlin available even without system zone data
)

func berlinTime(t *testing.T, year int, month time.Month, day, hour, min, sec int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("load Europe/Berlin: %v", err)
	}
	return time.Date(year, month, day, hour, min, sec, 0, loc)
}

func TestCurrentAnnouncementDefaultWindow(t *testing.T) {
	t.Setenv(awardAnnouncementStartEnv, "")
	t.Setenv(awardAnnouncementEndEnv, "")

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"one second before the embargo (Berlin)", berlinTime(t, 2026, 11, 13, 22, 29, 59), false},
		{"exactly at the embargo (Berlin)", berlinTime(t, 2026, 11, 13, 22, 30, 0), true},
		{"same instant expressed in UTC", time.Date(2026, 11, 13, 21, 30, 0, 0, time.UTC), true},
		{"22:30 UTC is already shown", time.Date(2026, 11, 13, 22, 30, 0, 0, time.UTC), true},
		{"21:29:59 UTC is 22:29:59 Berlin, hidden", time.Date(2026, 11, 13, 21, 29, 59, 0, time.UTC), false},
		{"weeks later", berlinTime(t, 2026, 12, 1, 12, 0, 0), true},
		{"one second before the end", berlinTime(t, 2026, 12, 13, 23, 59, 59), true},
		{"at the end", berlinTime(t, 2026, 12, 14, 0, 0, 0), false},
		{"long after", berlinTime(t, 2027, 1, 1, 0, 0, 0), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := currentAnnouncement(tt.now)
			if got.Active != tt.want {
				t.Fatalf("active = %v, want %v", got.Active, tt.want)
			}
			if !got.Active && (got.ID != "" || got.Link != "" || got.StartsAt != nil) {
				t.Fatalf("inactive response leaks details: %+v", got)
			}
			if got.Active && (got.ID != awardAnnouncementID || got.Link != awardAnnouncementLink) {
				t.Fatalf("active response incomplete: %+v", got)
			}
		})
	}
}

func TestCurrentAnnouncementEnvOverride(t *testing.T) {
	t.Setenv(awardAnnouncementStartEnv, "2026-11-01T10:00:00+01:00")
	t.Setenv(awardAnnouncementEndEnv, "2026-11-02T10:00:00+01:00")

	if !currentAnnouncement(berlinTime(t, 2026, 11, 1, 12, 0, 0)).Active {
		t.Fatal("override window should be active")
	}
	if currentAnnouncement(berlinTime(t, 2026, 11, 13, 23, 0, 0)).Active {
		t.Fatal("override end should hide the notice")
	}
}

func TestCurrentAnnouncementInvalidEnvFallsBack(t *testing.T) {
	t.Setenv(awardAnnouncementStartEnv, "13.11.2026 22:30")
	t.Setenv(awardAnnouncementEndEnv, "")

	if currentAnnouncement(berlinTime(t, 2026, 11, 13, 22, 0, 0)).Active {
		t.Fatal("an unparsable start must not switch the notice on early")
	}
	if !currentAnnouncement(berlinTime(t, 2026, 11, 13, 22, 30, 0)).Active {
		t.Fatal("default start should apply")
	}
}

// A window across the switch to summer time: the end is written with the
// summer offset (+02:00). Checks that offsets, not wall-clock text, decide.
func TestCurrentAnnouncementAcrossDST(t *testing.T) {
	t.Setenv(awardAnnouncementStartEnv, "2026-03-29T01:30:00+01:00")
	t.Setenv(awardAnnouncementEndEnv, "2026-03-29T03:30:00+02:00")

	// 01:30 CET -> 03:30 CEST is one real hour, because 02:00 jumps to 03:00.
	if !currentAnnouncement(berlinTime(t, 2026, 3, 29, 1, 59, 59)).Active {
		t.Fatal("just before the jump should be active")
	}
	if !currentAnnouncement(berlinTime(t, 2026, 3, 29, 3, 29, 59)).Active {
		t.Fatal("just before the end (summer time) should be active")
	}
	if currentAnnouncement(berlinTime(t, 2026, 3, 29, 3, 30, 0)).Active {
		t.Fatal("at the end (summer time) should be inactive")
	}
}
