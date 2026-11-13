// package: db / database access and repository layer
// type:    data
// job:     records anonymous play statistics (session start, AI turns) and assembles the admin dashboard numbers.
// limits:  no SQL (-> db/sqlc, queries/stats.sql), no HTTP (-> api/routes/admin_stats.go).
package db

import (
	"context"
	"time"

	sqlc "cgl/db/sqlc"
	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
)

// statsMonths is how many calendar months the dashboard shows (including the current one).
const statsMonths = 12

var statsLocation = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// RecordPlayStart notes a started session for the statistics. Best effort: errors are only logged.
func RecordPlayStart(ctx context.Context, sessionID, gameID, userID uuid.UUID, share *obj.GameShare) {
	params := sqlc.RecordPlayStartParams{SessionID: sessionID, GameID: gameID, UserID: userID, ViaShare: share != nil}
	if share != nil {
		if share.InstitutionID != nil {
			params.ShareInstitutionID = uuid.NullUUID{UUID: *share.InstitutionID, Valid: true}
		}
		if share.WorkshopID != nil {
			params.ShareWorkshopID = uuid.NullUUID{UUID: *share.WorkshopID, Valid: true}
		}
	}
	if err := queries().RecordPlayStart(ctx, params); err != nil {
		log.Warn("failed to record play start for statistics", "session_id", sessionID, "error", err)
	}
}

// RecordPlayTurn counts one AI response of a session. Best effort.
func RecordPlayTurn(ctx context.Context, sessionID uuid.UUID) {
	if err := queries().RecordPlayTurn(ctx, sessionID); err != nil {
		log.Warn("failed to record play turn for statistics", "session_id", sessionID, "error", err)
	}
}

// statsMonthKeys returns the last statsMonths months ("YYYY-MM", oldest first) and the start of the first.
func statsMonthKeys(now time.Time) ([]string, time.Time) {
	local := now.In(statsLocation)
	first := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, statsLocation).AddDate(0, -(statsMonths - 1), 0)
	keys := make([]string, statsMonths)
	for i := range keys {
		keys[i] = first.AddDate(0, i, 0).Format("2006-01")
	}
	return keys, first
}

func emptyMonths(keys []string) []obj.StatsMonth {
	m := make([]obj.StatsMonth, len(keys))
	for i, k := range keys {
		m[i].Month = k
	}
	return m
}

func monthIndex(keys []string) map[string]int {
	idx := make(map[string]int, len(keys))
	for i, k := range keys {
		idx[k] = i
	}
	return idx
}

// GetAdminStats assembles the dashboard numbers. Admin only.
func GetAdminStats(ctx context.Context, adminID uuid.UUID) (*obj.AdminStats, error) {
	if err := requireAdmin(ctx, adminID); err != nil {
		return nil, err
	}
	now := time.Now()
	keys, since := statsMonthKeys(now)
	idx := monthIndex(keys)
	since7, since30 := now.AddDate(0, 0, -7), now.AddDate(0, 0, -30)
	// Games and plays older than the month window can still fall into "last 30 days".
	if since30.Before(since) {
		since = since30
	}

	q := queries()
	stats := &obj.AdminStats{GeneratedAt: now, Months: emptyMonths(keys)}

	counts, err := q.StatsInstitutionCounts(ctx)
	if err != nil {
		return nil, obj.ErrServerError("failed to count organisations")
	}
	stats.Institutions, stats.InstitutionsLocked = int(counts.Total), int(counts.Locked)

	if first, err := q.StatsPlayStatSince(ctx); err == nil {
		stats.PlaysRecordedSince = first
	}

	perMonth, err := q.StatsInstitutionsPerMonth(ctx, since)
	if err != nil {
		return nil, obj.ErrServerError("failed to count new organisations")
	}
	for _, r := range perMonth {
		if i, ok := idx[r.Month]; ok {
			stats.Months[i].NewInstitutions = int(r.N)
		}
	}

	overview, err := q.StatsInstitutionOverview(ctx)
	if err != nil {
		return nil, obj.ErrServerError("failed to load organisations")
	}
	byID := make(map[uuid.UUID]*obj.InstitutionStats, len(overview))
	stats.PerInstitution = make([]obj.InstitutionStats, len(overview))
	for i, r := range overview {
		s := obj.InstitutionStats{
			ID: r.ID, Name: r.Name, City: nullStringToPtr(r.City), CreatedAt: r.CreatedAt,
			Locked: r.DisabledAt.Valid, Heads: int(r.Heads), Staff: int(r.Staff),
			Individuals: int(r.Individuals), Participants: int(r.Participants),
			Workshops: int(r.Workshops), ActiveWorkshops: int(r.ActiveWorkshops), Games: int(r.Games),
			Months: emptyMonths(keys),
		}
		if r.LastActivity.After(time.Unix(0, 0)) {
			t := r.LastActivity
			s.LastActivity = &t
		}
		if status, err := InstitutionDeclarationStatusFor(ctx, r.ID); err == nil && status.Pending {
			s.ConsentPending = true
			stats.ConsentPending++
		}
		stats.PerInstitution[i] = s
		byID[r.ID] = &stats.PerInstitution[i]
	}

	add := func(p *obj.StatsPeriod, last7, last30 bool, n int) {
		if last7 {
			p.Last7 += n
		}
		if last30 {
			p.Last30 += n
		}
	}

	games, err := q.StatsGamesPerInstitutionMonth(ctx, sqlc.StatsGamesPerInstitutionMonthParams{Since7: since7, Since30: since30, Since: since})
	if err != nil {
		return nil, obj.ErrServerError("failed to count games")
	}
	for _, r := range games {
		n := int(r.N)
		add(&stats.All.NewGames, r.Last7, r.Last30, n)
		i, inWindow := idx[r.Month]
		if inWindow {
			stats.Months[i].NewGames += n
		}
		if inst, ok := byID[r.InstitutionID]; ok {
			add(&inst.NewGames, r.Last7, r.Last30, n)
			if inWindow {
				inst.Months[i].NewGames += n
			}
		} else {
			add(&stats.OutsideOrganisations.NewGames, r.Last7, r.Last30, n)
		}
	}

	plays, err := q.StatsPlaysPerInstitutionMonth(ctx, sqlc.StatsPlaysPerInstitutionMonthParams{Since7: since7, Since30: since30, Since: since})
	if err != nil {
		return nil, obj.ErrServerError("failed to count plays")
	}
	for _, r := range plays {
		sessions, turns := int(r.Sessions), int(r.Turns)
		add(&stats.All.Sessions, r.Last7, r.Last30, sessions)
		add(&stats.All.Turns, r.Last7, r.Last30, turns)
		i, inWindow := idx[r.Month]
		if inWindow {
			stats.Months[i].Sessions += sessions
			stats.Months[i].Turns += turns
		}
		var inst *obj.InstitutionStats
		if r.InstitutionID.Valid {
			inst = byID[r.InstitutionID.UUID]
		}
		if inst != nil {
			add(&inst.Sessions, r.Last7, r.Last30, sessions)
			add(&inst.Turns, r.Last7, r.Last30, turns)
			if inWindow {
				inst.Months[i].Sessions += sessions
				inst.Months[i].Turns += turns
			}
		} else {
			add(&stats.OutsideOrganisations.Sessions, r.Last7, r.Last30, sessions)
			add(&stats.OutsideOrganisations.Turns, r.Last7, r.Last30, turns)
		}
	}

	funnel, err := q.StatsRegistrationFunnel(ctx, now.AddDate(0, 0, -90))
	if err != nil {
		return nil, obj.ErrServerError("failed to count registrations")
	}
	stats.Funnel = obj.RegistrationFunnel{
		Submitted: int(funnel.Submitted), Confirmed: int(funnel.Confirmed), WithWorkshop: int(funnel.WithWorkshop),
	}
	return stats, nil
}
