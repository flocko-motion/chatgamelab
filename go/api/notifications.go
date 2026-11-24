// package: api / schemaf endpoints
// type:    logic
// job:     the caller's notifications (the bell): reads them, acknowledges one or all
// limits:  decides nothing: the core does (core/notification.go)
package api

import (
	"context"
	"time"

	"cgl/api/wire"
	"cgl/core"

	schemafapi "github.com/flocko-motion/schemaf/api"
	"github.com/google/uuid"
)

// NotificationsResp is the caller's current notifications, newest first.
type NotificationsResp struct {
	Notifications []wire.NotificationView `json:"notifications"`
}

func notificationsResp(ns []core.Notification, err error) (NotificationsResp, error) {
	return NotificationsResp{Notifications: wire.NotificationViewsOf(ns)}, err
}

// NotificationsReq asks for nothing more: the notifications are the
// caller's.
type NotificationsReq struct{}

// ListNotificationsEndpoint answers the caller's notifications: the
// important facts that are true now and wait for their acknowledgement.
type ListNotificationsEndpoint struct{}

func (ListNotificationsEndpoint) Method() string             { return "GET" }
func (ListNotificationsEndpoint) Path() string               { return "/api/notifications" }
func (ListNotificationsEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ListNotificationsEndpoint) Handle(ctx context.Context, _ NotificationsReq) (NotificationsResp, error) {
	return notificationsResp(core.Notifications(ctx))
}

// AcknowledgeNotificationReq names the notification the caller
// acknowledges: its type and, for a type about one thing, its subject.
type AcknowledgeNotificationReq struct {
	Type    core.NotificationType `json:"type"`
	Subject *uuid.UUID            `json:"subject,omitempty"`
	// At is the time of the entry the caller saw (its at); facts after it
	// stay news. Without it, everything up to now is acknowledged.
	At *time.Time `json:"at,omitempty"`
}

// AcknowledgeNotificationEndpoint acknowledges one of the caller's
// notifications and answers those that remain.
type AcknowledgeNotificationEndpoint struct{}

func (AcknowledgeNotificationEndpoint) Method() string { return "POST" }
func (AcknowledgeNotificationEndpoint) Path() string   { return "/api/notifications/acknowledge" }
func (AcknowledgeNotificationEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (AcknowledgeNotificationEndpoint) Handle(ctx context.Context, req AcknowledgeNotificationReq) (NotificationsResp, error) {
	if err := core.AcknowledgeNotification(ctx, req.Type, req.Subject, req.At); err != nil {
		return NotificationsResp{}, err
	}
	return notificationsResp(core.Notifications(ctx))
}

// AcknowledgeAllNotificationsEndpoint acknowledges every notification the
// caller has, each up to the entry they could see; it answers those that
// remain.
type AcknowledgeAllNotificationsEndpoint struct{}

func (AcknowledgeAllNotificationsEndpoint) Method() string { return "POST" }
func (AcknowledgeAllNotificationsEndpoint) Path() string   { return "/api/notifications/acknowledge-all" }
func (AcknowledgeAllNotificationsEndpoint) Auth() schemafapi.AuthLevel {
	return schemafapi.AuthRequired
}
func (AcknowledgeAllNotificationsEndpoint) Handle(ctx context.Context, _ NotificationsReq) (NotificationsResp, error) {
	if err := core.AcknowledgeAllNotifications(ctx); err != nil {
		return NotificationsResp{}, err
	}
	return notificationsResp(core.Notifications(ctx))
}
