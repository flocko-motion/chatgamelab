package testing

import (
	"net/http"

	"cgl/obj"
	"cgl/testing/testutil"
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

// postShareView calls the play page's view counter with the given user agent; returns the HTTP status.
func postShareView(token, userAgent string) (int, error) {
	req, err := http.NewRequest(http.MethodPost, testutil.TestServerURL+"/api/play/"+token+"/view", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

func findSharedGame(stats obj.AdminStats, gameID string) *obj.SharedGameStats {
	for i := range stats.Shares.Games {
		if stats.Shares.Games[i].GameID.String() == gameID {
			return &stats.Shares.Games[i]
		}
	}
	return nil
}

// TestAdminStatsCountShareViews: page views of a share link from browsers are counted, link previews
// and scripts are not; a game started from the link counts as start of that game.
func (s *InstitutionRegistrationSuite) TestAdminStatsCountShareViews() {
	admin := s.DevUser()
	owner := s.CreateUser("share-views")
	keyShare := Must(owner.AddApiKey("mock-share-views", "Share Views Key", "mock"))
	game := Must(owner.UploadGame("alien-first-contact"))
	share := Must(owner.CreateGameShare(game.ID.String(), keyShare.ID.String(), nil))

	var before obj.AdminStats
	MustSucceed(admin.Get("admin/stats", &before))

	browser := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15"
	for _, ua := range []string{browser, browser, "WhatsApp/2.24 facebookexternalhit/1.1", "Mozilla/5.0 (compatible; Googlebot/2.1)", "curl/8.4.0", ""} {
		status, err := postShareView(share.Token, ua)
		s.Require().NoError(err)
		s.Equal(http.StatusNoContent, status, "user agent %q", ua)
	}
	status, err := postShareView("no-such-token", browser)
	s.Require().NoError(err)
	s.GreaterOrEqual(status, 400, "unknown token is rejected")

	_, _, err = s.Public().GuestCreateSessionWithStream(share.Token)
	s.Require().NoError(err)

	var after obj.AdminStats
	MustSucceed(admin.Get("admin/stats", &after))

	g := findSharedGame(after, game.ID.String())
	s.Require().NotNil(g, "game with share views is listed")
	s.Equal(game.Name, g.Name)
	s.Equal(2, g.Views.Last7, "only the two browser views count")
	s.Equal(2, g.Views.Last30)
	s.Equal(1, g.Starts.Last7)
	s.Equal(1, g.Starts.Last30)
	s.False(g.Deleted)

	s.Equal(before.Shares.Views.Last7+2, after.Shares.Views.Last7)
	s.Equal(before.Shares.Starts.Last30+1, after.Shares.Starts.Last30)
	s.Equal(before.Months[11].ShareViews+2, after.Months[11].ShareViews)
	s.False(after.Shares.ViewsRecordedSince.IsZero())
}
