// package: core / notifications
// type:    logic
// job:     the caller's notifications (the bell): which types they hold, what is news, acknowledging
// limits:  the facts themselves belong to their entities (organisation.go, verified_page.go)
package core

import (
	"context"
	"slices"
	"time"

	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"

	"github.com/google/uuid"
)

// A notification is an important fact that is true now and waits for the
// user's acknowledgement (#578). It is no inbox: nothing is stored about
// the facts. Each type is a rule here that reads the current data and
// compares it with the user's last acknowledgement of that type; without
// one, their starting point is when they got their role. An entry goes when
// it is acknowledged or no longer true, and the next new fact starts a fresh
// one. Read state is personal, and a caller only ever gets their own.

// NotificationType is a kind of notification; the web app writes its text.
type NotificationType string

const (
	// NotificationNewOrganisations: organisations created since, for whoever
	// lists every organisation.
	NotificationNewOrganisations NotificationType = "newOrganisations"
	// NotificationPagesWaiting: verified pages asked for since and still
	// waiting, for whoever approves them.
	NotificationPagesWaiting NotificationType = "pagesWaiting"
	// NotificationPageApproved: an admin approved the own organisation's
	// verified page since, for whoever lists it.
	NotificationPageApproved NotificationType = "pageApproved"
	// NotificationPageTurnedDown: an admin turned the own organisation's
	// verified page down or withdrew its approval since, for whoever lists
	// it. A change of its site by the organisation is no turning down.
	NotificationPageTurnedDown NotificationType = "pageTurnedDown"
)

// Notification is one entry of the caller's bell.
type Notification struct {
	Type    NotificationType
	Subject *uuid.UUID // the one thing it is about, for a type about one thing
	Count   int        // the facts it folds, at least 1
	At      time.Time  // the newest of them: acknowledging up to it leaves later ones news
	// OrganisationID is the organisation it is about, for a type about the
	// caller's own: where it leads.
	OrganisationID *uuid.UUID
}

// notificationRule is one type: whether the caller holds it, and what is
// news to them, read for user from where their notifications of it start.
type notificationRule struct {
	typ   NotificationType
	holds func(ctx context.Context) bool
	news  func(ctx context.Context, user uuid.UUID) ([]Notification, error)
}

// notificationRules are the types, each held through the grant that opens
// what it tells about.
var notificationRules = []notificationRule{
	{NotificationNewOrganisations, mayHearOfNewOrganisations, newOrganisationsNews},
	{NotificationPagesWaiting, mayApproveVerifiedPages, pagesWaitingNews},
	{NotificationPageApproved, mayHearOfOwnVerifiedPage, pageApprovedNews},
	{NotificationPageTurnedDown, mayHearOfOwnVerifiedPage, pageTurnedDownNews},
}

// mayHearOfNewOrganisations: whoever lists every organisation is told of
// new ones, {List, OrganisationCard, Everywhere}.
func mayHearOfNewOrganisations(ctx context.Context) bool {
	return granted(ctx, rules.List, rules.OrganisationCard, rules.Everywhere)
}

// mayHearOfOwnVerifiedPage: whoever lists their organisation's verified
// page is told how admins decided it, {List, VerifiedPage,
// OfOwnOrganisation}.
func mayHearOfOwnVerifiedPage(ctx context.Context) bool {
	return from(ctx).Organisation() != nil && granted(ctx, rules.List, rules.VerifiedPage, rules.OfOwnOrganisation)
}

// Notifications are the caller's current notifications, newest first: an
// entry for every type they hold that has news.
func Notifications(ctx context.Context) ([]Notification, error) {
	user := from(ctx).Account()
	out := []Notification{}
	if user == nil {
		return out, nil
	}
	for _, r := range notificationRules {
		if !r.holds(ctx) {
			continue
		}
		ns, err := r.news(ctx, *user)
		if err != nil {
			return nil, err
		}
		out = append(out, ns...)
	}
	slices.SortStableFunc(out, func(a, b Notification) int { return b.At.Compare(a.At) })
	return out, nil
}

// AcknowledgeNotification acknowledges the caller's notifications of type
// t (about subject, for a type about one thing) up to upTo, at most now;
// nil is now. A type the caller does not hold is not found.
func AcknowledgeNotification(ctx context.Context, t NotificationType, subject *uuid.UUID, upTo *time.Time) error {
	i := slices.IndexFunc(notificationRules, func(r notificationRule) bool { return r.typ == t })
	if i < 0 {
		return refuse("unknown notification type %q", t)
	}
	user := from(ctx).Account()
	if user == nil || !notificationRules[i].holds(ctx) {
		return ErrNotFound
	}
	if subject != nil {
		return refuse("notifications of type %q are about no one thing: leave out the subject", t)
	}
	return store.Queries().V2AcknowledgeNotification(ctx, db.V2AcknowledgeNotificationParams{
		UserID: *user, Type: string(t), Subject: nullUUID(subject), UpTo: sqlTime(upTo),
	})
}

// AcknowledgeAllNotifications acknowledges every notification the caller
// has, each up to its entry: a fact arriving meanwhile stays news.
func AcknowledgeAllNotifications(ctx context.Context) error {
	ns, err := Notifications(ctx)
	if err != nil {
		return err
	}
	for _, n := range ns {
		if err := AcknowledgeNotification(ctx, n.Type, n.Subject, &n.At); err != nil {
			return err
		}
	}
	return nil
}

func newOrganisationsNews(ctx context.Context, user uuid.UUID) ([]Notification, error) {
	r, err := store.Queries().V2NewOrganisationsNotice(ctx, db.V2NewOrganisationsNoticeParams{
		UserID: user, Type: string(NotificationNewOrganisations),
	})
	return folded(NotificationNewOrganisations, int(r.Count), r.At, err)
}

func pagesWaitingNews(ctx context.Context, user uuid.UUID) ([]Notification, error) {
	r, err := store.Queries().V2PagesWaitingNotice(ctx, db.V2PagesWaitingNoticeParams{
		UserID: user, Type: string(NotificationPagesWaiting),
	})
	return folded(NotificationPagesWaiting, int(r.Count), r.At, err)
}

func pageApprovedNews(ctx context.Context, user uuid.UUID) ([]Notification, error) {
	org := from(ctx).Organisation()
	at, err := store.Queries().V2PageApprovedNotice(ctx, db.V2PageApprovedNoticeParams{
		UserID: user, Type: string(NotificationPageApproved), InstitutionID: *org,
	})
	return aboutOwnOrganisation(NotificationPageApproved, *org, at, err)
}

func pageTurnedDownNews(ctx context.Context, user uuid.UUID) ([]Notification, error) {
	org := from(ctx).Organisation()
	at, err := store.Queries().V2PageTurnedDownNotice(ctx, db.V2PageTurnedDownNoticeParams{
		UserID: user, Type: string(NotificationPageTurnedDown), InstitutionID: *org,
	})
	return aboutOwnOrganisation(NotificationPageTurnedDown, *org, at, err)
}

// folded is one entry folding count facts, the newest at at; none without
// facts.
func folded(t NotificationType, count int, at time.Time, err error) ([]Notification, error) {
	if err != nil || count == 0 {
		return nil, err
	}
	return []Notification{{Type: t, Count: count, At: at}}, nil
}

// aboutOwnOrganisation is an entry about the caller's organisation org, if
// there is news: its time, at most one.
func aboutOwnOrganisation(t NotificationType, org uuid.UUID, at []time.Time, err error) ([]Notification, error) {
	if err != nil || len(at) == 0 {
		return nil, err
	}
	return []Notification{{Type: t, Count: 1, At: at[0], OrganisationID: &org}}, nil
}
