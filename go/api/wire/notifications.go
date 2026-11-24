// package: api/wire / notification views
// type:    logic
// job:     the caller's notifications (the bell) on the wire
// limits:  converts the core's types; decides nothing (-> core/notification.go)
package wire

import (
	"time"

	"cgl/core"

	"github.com/google/uuid"
)

// NotificationView is one entry of the caller's bell: a fact that is true
// now and waits for their acknowledgement. The web app writes its text from
// type and count.
type NotificationView struct {
	Type core.NotificationType `json:"type"`
	// Subject is the one thing the entry is about, for a type about one
	// thing; acknowledging names it.
	Subject *uuid.UUID `json:"subject,omitempty"`
	// Count is how many facts the entry folds, at least 1.
	Count int `json:"count"`
	// At is the time of the newest fact it folds; acknowledging up to it
	// leaves later facts news.
	At time.Time `json:"at"`
	// OrganisationID is the organisation it is about, for a type about the
	// caller's organisation: where clicking it leads.
	OrganisationID *uuid.UUID `json:"organisationId,omitempty"`
}

// NotificationViewsOf converts the caller's notifications.
func NotificationViewsOf(ns []core.Notification) []NotificationView {
	out := make([]NotificationView, len(ns))
	for i, n := range ns {
		out[i] = NotificationView{Type: n.Type, Subject: n.Subject, Count: n.Count, At: n.At, OrganisationID: n.OrganisationID}
	}
	return out
}
