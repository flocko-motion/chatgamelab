// package: routes / admin statistics handler
// type:    logic
// job:     serves the numbers of the admin dashboard.
// limits:  no route registration (-> router.go); assembly in db/stats.go.
package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
)

// GetAdminStats godoc
//
//	@Summary		Admin statistics
//	@Description	Admin only. Organisations, new games, started sessions and AI turns: totals, last 7 and 30 days, per month (last 12 calendar months, Europe/Berlin) and per organisation. Sessions and AI turns are counted from playsRecordedSince on, because game sessions themselves are deleted 48 h after the last activity.
//	@Tags			admin
//	@Produce		json
//	@Success		200	{object}	obj.AdminStats
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/admin/stats [get]
func GetAdminStats(w http.ResponseWriter, r *http.Request) {
	user := httpx.UserFromRequest(r)
	stats, err := db.GetAdminStats(r.Context(), user.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, stats)
}
