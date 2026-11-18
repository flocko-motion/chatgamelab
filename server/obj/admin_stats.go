// package: obj / admin statistics types
// type:    data
// job:     defines the admin dashboard statistics: totals, last 7/30 days, per month and per organisation.
// limits:  no queries (-> db/stats.go), no HTTP (-> api/routes/admin_stats.go).
package obj

import (
	"time"

	"github.com/google/uuid"
)

// StatsPeriod counts something in the last 7 and the last 30 days.
type StatsPeriod struct {
	Last7  int `json:"last7"`
	Last30 int `json:"last30"`
}

// StatsMonth is one calendar month (Europe/Berlin), "YYYY-MM".
type StatsMonth struct {
	Month           string `json:"month"`
	NewGames        int    `json:"newGames"`
	Sessions        int    `json:"sessions"`
	Turns           int    `json:"turns"`
	NewInstitutions int    `json:"newInstitutions,omitempty"`
	ShareViews      int    `json:"shareViews"`
}

// InstitutionStats are the numbers of one organisation.
type InstitutionStats struct {
	ID              uuid.UUID    `json:"id"`
	Name            string       `json:"name"`
	City            *string      `json:"city,omitempty"`
	CreatedAt       time.Time    `json:"createdAt"`
	Locked          bool         `json:"locked"`
	ConsentPending  bool         `json:"consentPending"`
	Heads           int          `json:"heads"`
	Staff           int          `json:"staff"`
	Individuals     int          `json:"individuals"`
	Participants    int          `json:"participants"`
	Workshops       int          `json:"workshops"`
	ActiveWorkshops int          `json:"activeWorkshops"`
	Games           int          `json:"games"`
	NewGames        StatsPeriod  `json:"newGames"`
	Sessions        StatsPeriod  `json:"sessions"`
	Turns           StatsPeriod  `json:"turns"`
	LastActivity    *time.Time   `json:"lastActivity,omitempty"`
	Months          []StatsMonth `json:"months"`
}

// StatsActivity groups new games, started sessions and AI turns.
type StatsActivity struct {
	NewGames StatsPeriod `json:"newGames"`
	Sessions StatsPeriod `json:"sessions"`
	Turns    StatsPeriod `json:"turns"`
}

// RegistrationFunnel counts self-registrations of the last 90 days.
type RegistrationFunnel struct {
	Submitted    int `json:"submitted"`
	Confirmed    int `json:"confirmed"`
	WithWorkshop int `json:"withWorkshop"`
}

// SharedGameStats compares page views of a game's share links with the plays started there.
type SharedGameStats struct {
	GameID  uuid.UUID   `json:"gameId"`
	Name    string      `json:"name"`
	Deleted bool        `json:"deleted"`
	Views   StatsPeriod `json:"views"`
	Starts  StatsPeriod `json:"starts"`
}

// ShareStats are page views of share links (/play/{token}) and the plays started from them.
type ShareStats struct {
	// ViewsRecordedSince: page views are counted from this moment on.
	ViewsRecordedSince time.Time         `json:"viewsRecordedSince"`
	Views              StatsPeriod       `json:"views"`
	Starts             StatsPeriod       `json:"starts"`
	Games              []SharedGameStats `json:"games"`
}

// AdminStats is everything the admin dashboard shows.
type AdminStats struct {
	GeneratedAt time.Time `json:"generatedAt"`
	// PlaysRecordedSince: sessions and AI turns are counted from this moment on
	// (game sessions themselves are deleted 48 h after the last activity).
	PlaysRecordedSince   time.Time          `json:"playsRecordedSince"`
	Institutions         int                `json:"institutions"`
	InstitutionsLocked   int                `json:"institutionsLocked"`
	ConsentPending       int                `json:"consentPending"`
	All                  StatsActivity      `json:"all"`
	OutsideOrganisations StatsActivity      `json:"outsideOrganisations"`
	Months               []StatsMonth       `json:"months"`
	Funnel               RegistrationFunnel `json:"funnel"`
	Shares               ShareStats         `json:"shares"`
	PerInstitution       []InstitutionStats `json:"perInstitution"`
}
