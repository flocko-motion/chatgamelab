package testing

import (
	"cgl/obj"
)

// findInstitutionStats returns the dashboard row of one organisation.
func findInstitutionStats(stats obj.AdminStats, name string) *obj.InstitutionStats {
	for i := range stats.PerInstitution {
		if stats.PerInstitution[i].Name == name {
			return &stats.PerInstitution[i]
		}
	}
	return nil
}

// TestAdminStatsCountPlaysPerOrganisation: a game played by the head of a new organisation shows up
// as new game, started session and AI turn of that organisation; only admins see the numbers.
func (s *InstitutionRegistrationSuite) TestAdminStatsCountPlaysPerOrganisation() {
	admin := s.DevUser()
	head, req, res := s.register("statistik@jz-test.de")

	var before obj.AdminStats
	MustSucceed(admin.Get("admin/stats", &before))

	key := Must(head.AddApiKey("mock-stats", "Stats Key", "mock"))
	Must(head.SetDefaultApiKey(key.ID.String()))
	game := Must(head.UploadGame("alien-first-contact"))
	_, _, err := head.CreateGameSessionWithStream(game.ID.String())
	s.Require().NoError(err)

	var after obj.AdminStats
	MustSucceed(admin.Get("admin/stats", &after))

	inst := findInstitutionStats(after, req.InstitutionName)
	s.Require().NotNil(inst, "new organisation is listed")
	s.Equal(res.Institution.ID, inst.ID)
	s.Equal(1, inst.Heads)
	s.Equal(1, inst.Games)
	s.Equal(1, inst.NewGames.Last7)
	s.Equal(1, inst.Sessions.Last7)
	s.Equal(1, inst.Sessions.Last30)
	s.GreaterOrEqual(inst.Turns.Last7, 1, "the opening scene counts as AI turn")
	s.NotNil(inst.LastActivity)
	s.False(inst.ConsentPending, "consent was given at registration")
	s.Len(inst.Months, 12)
	s.Equal(1, inst.Months[11].Sessions, "current month is last")

	// Totals grew by the same amount.
	s.Equal(before.All.Sessions.Last7+1, after.All.Sessions.Last7)
	s.Equal(before.All.NewGames.Last30+1, after.All.NewGames.Last30)
	s.Equal(before.Institutions, after.Institutions)
	s.False(after.PlaysRecordedSince.IsZero())

	// Only admins.
	err = head.Get("admin/stats", nil)
	s.Require().Error(err)
	s.Contains(err.Error(), "403")
}
