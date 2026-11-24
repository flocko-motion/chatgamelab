// package: testutil / the words for notifications
// type:    logic
// job:     the caller's notifications (the bell): reading and acknowledging them
// limits:  notification words only; what causes them lives in orgs.go and embed.go
package testutil

import (
	"cgl/apiclient"
)

// Notification is one entry of the caller's bell.
type Notification = apiclient.NotificationView

// The notification types.
const (
	NewOrganisations = apiclient.NotificationTypeNewOrganisations
	PagesWaiting     = apiclient.NotificationTypePagesWaiting
	PageApproved     = apiclient.NotificationTypePageApproved
	PageTurnedDown   = apiclient.NotificationTypePageTurnedDown
)

// Notifications are the caller's current notifications.
func (c caller) Notifications() ([]Notification, error) {
	resp, err := decode[apiclient.NotificationsResp](c.api.GetApiNotificationsWithResponse(ctx()))
	return resp.Notifications, err
}

// Notification is the caller's current notification of type t; nil while
// there is none.
func (c caller) Notification(t apiclient.NotificationType) (*Notification, error) {
	all, err := c.Notifications()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].Type == t {
			return &all[i], nil
		}
	}
	return nil, nil
}

// Acknowledge acknowledges notification n, as clicking it does.
func (c caller) Acknowledge(n Notification) error {
	return done(c.api.PostApiNotificationsAcknowledgeWithResponse(ctx(), apiclient.AcknowledgeNotificationReq{
		Type: n.Type, Subject: n.Subject, At: &n.At,
	}))
}

// AcknowledgeType acknowledges the caller's notification of type t up to
// now, whether they have one or not.
func (c caller) AcknowledgeType(t apiclient.NotificationType) error {
	return done(c.api.PostApiNotificationsAcknowledgeWithResponse(ctx(), apiclient.AcknowledgeNotificationReq{Type: t}))
}

// AcknowledgeAll acknowledges every notification the caller has.
func (c caller) AcknowledgeAll() error {
	return done(c.api.PostApiNotificationsAcknowledgeAllWithResponse(ctx()))
}
