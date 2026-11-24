// package: core / lists
// type:    logic
// job:     every entity list as one core.List: its spec and its one query
// limits:  the list contract itself (-> list.go)
package core

import (
	"context"
	"slices"
	"strings"
	"time"

	"cgl/obj"

	"github.com/google/uuid"
)

// The lists of the core's entities besides games (GameCards, game.go). Each
// loads what its gates let through and answers the query from it
// (pageItems): their gated sets are small. Search and order use only what
// the caller may see of an item.

// keepAll is the filter of a list whose filters its gated query applied.
func keepAll[T any](string) func(T) bool { return func(T) bool { return true } }

// ── Users ──────────────────────────────────────────────────────────────────

// Users is the admins' user list; noGuests leaves out guests and users
// without a role.
var Users Lister[*UserCard] = users{}

type users struct{}

func (users) Spec() ListSpec {
	return ListSpec{
		Filters:     []string{"noGuests"},
		Sorts:       []string{"name", "email", "role", "organisation", "createdAt"},
		DefaultSort: "name",
		DefaultDir:  "asc",
	}
}

func (users) Query(ctx context.Context, q ListQuery) (Page[*UserCard], error) {
	all, err := ListUsers(ctx)
	if err != nil {
		return Page[*UserCard]{}, err
	}
	return pageItems(all, q, userRules(map[string]func(*UserCard) bool{
		"noGuests": func(u *UserCard) bool { r := userRole(u); return r != "" && r != string(obj.RoleGuest) },
	})), nil
}

// Members is organisation id's member list: {List, UserCard}, narrowed to
// the organisation. Orders name (default), email and role.
func Members(id uuid.UUID) Lister[*UserCard] { return members{id} }

type members struct{ organisation uuid.UUID }

func (members) Spec() ListSpec {
	return ListSpec{Sorts: []string{"name", "email", "role"}, DefaultSort: "name", DefaultDir: "asc"}
}

func (m members) Query(ctx context.Context, q ListQuery) (Page[*UserCard], error) {
	all, err := ListMembers(ctx, m.organisation)
	if err != nil {
		return Page[*UserCard]{}, err
	}
	return pageItems(all, q, userRules(nil)), nil
}

// userRules search and order users by what the caller may see of them.
func userRules(filters map[string]func(*UserCard) bool) itemRules[*UserCard] {
	return itemRules[*UserCard]{
		search: func(u *UserCard) []string {
			return []string{u.Name(), userEmail(u), userRole(u), userOrganisation(u)}
		},
		filter: func(name string) func(*UserCard) bool { return filters[name] },
		sorts: map[string]func(a, b *UserCard) int{
			"name":         byText(func(u *UserCard) string { return u.Name() }),
			"email":        byText(userEmail),
			"role":         byText(userRole),
			"organisation": byText(userOrganisation),
			"createdAt":    byValue(func(u *UserCard) int64 { return userCreatedAt(u).UnixNano() }),
		},
	}
}

func userRole(u *UserCard) string {
	if a, err := u.AsUserAccount(); err == nil {
		return a.Role()
	}
	return ""
}

func userOrganisation(u *UserCard) string {
	if a, err := u.AsUserAccount(); err == nil && a.OrganisationName() != nil {
		return *a.OrganisationName()
	}
	return ""
}

func userCreatedAt(u *UserCard) time.Time {
	if a, err := u.AsUserAccount(); err == nil {
		return a.CreatedAt()
	}
	return time.Time{}
}

func userEmail(u *UserCard) string {
	if c, err := u.AsUserContact(); err == nil && c.Email() != nil {
		return *c.Email()
	}
	return ""
}

// ── Organisations ──────────────────────────────────────────────────────────

// Organisations is the organisation list; recentlyRegistered keeps those
// registered lately.
var Organisations Lister[*OrganisationCard] = organisations{}

type organisations struct{}

func (organisations) Spec() ListSpec {
	return ListSpec{
		Filters:     []string{OrganisationFilterRecentlyRegistered},
		Sorts:       []string{"name", "createdAt", "registeredAt"},
		DefaultSort: "name",
		DefaultDir:  "asc",
	}
}

func (organisations) Query(ctx context.Context, q ListQuery) (Page[*OrganisationCard], error) {
	all, err := ListOrganisations(ctx, q.Filter)
	if err != nil {
		return Page[*OrganisationCard]{}, err
	}
	return pageItems(all, q, itemRules[*OrganisationCard]{
		search: func(o *OrganisationCard) []string {
			texts := []string{o.Name()}
			if d, err := o.AsOrganisationDetails(); err == nil {
				texts = append(texts, deref(d.Operator()), deref(d.City()))
			}
			for _, m := range o.Members() {
				if userRole(m) == string(obj.RoleHead) {
					texts = append(texts, m.Name(), userEmail(m))
				}
			}
			return texts
		},
		filter: keepAll[*OrganisationCard],
		sorts: map[string]func(a, b *OrganisationCard) int{
			"name":         byText(func(o *OrganisationCard) string { return o.Name() }),
			"createdAt":    byValue(func(o *OrganisationCard) int64 { return o.CreatedAt().UnixNano() }),
			"registeredAt": byValue(func(o *OrganisationCard) int64 { return organisationRegisteredAt(o).UnixNano() }),
		},
	}), nil
}

func organisationRegisteredAt(o *OrganisationCard) time.Time {
	if r, err := o.AsOrganisationRecord(); err == nil && r.Registration() != nil {
		return r.Registration().At
	}
	return time.Time{}
}

// ── Invites ────────────────────────────────────────────────────────────────

// Invites is the invite list narrowed by f, newest first.
func Invites(f InviteFilter) Lister[*InviteCard] { return invites{f} }

type invites struct{ filter InviteFilter }

func (invites) Spec() ListSpec {
	return ListSpec{
		Sorts:       []string{"role", "organisation", "status", "modifiedAt"},
		DefaultSort: "modifiedAt",
		DefaultDir:  "desc",
	}
}

func (l invites) Query(ctx context.Context, q ListQuery) (Page[*InviteCard], error) {
	all, err := ListInvites(ctx, l.filter)
	if err != nil {
		return Page[*InviteCard]{}, err
	}
	return pageItems(all, q, itemRules[*InviteCard]{
		search: func(i *InviteCard) []string {
			return []string{deref(i.InvitedEmail), i.Role, i.Status, i.OrganisationName, deref(i.WorkshopName)}
		},
		filter: keepAll[*InviteCard],
		sorts: map[string]func(a, b *InviteCard) int{
			"role":         byText(func(i *InviteCard) string { return i.Role }),
			"organisation": byText(func(i *InviteCard) string { return i.OrganisationName }),
			"status":       byText(func(i *InviteCard) string { return i.Status }),
			"modifiedAt":   byValue(func(i *InviteCard) int64 { return i.ModifiedAt.UnixNano() }),
		},
	}), nil
}

// ── Workshops and their participants ───────────────────────────────────────

// Workshops is organisation id's workshop list, or every one the caller may
// list without an id.
func Workshops(organisation *uuid.UUID) Lister[*WorkshopCard] { return workshops{organisation} }

type workshops struct{ organisation *uuid.UUID }

func (workshops) Spec() ListSpec {
	return ListSpec{
		Filters:     []string{"active"},
		Sorts:       []string{"name", "createdAt", "modifiedAt", "participantCount"},
		DefaultSort: "createdAt",
		DefaultDir:  "desc",
	}
}

func (l workshops) Query(ctx context.Context, q ListQuery) (Page[*WorkshopCard], error) {
	all, err := ListWorkshops(ctx, WorkshopQuery{
		Organisation: l.organisation,
		Search:       q.Search,
		ActiveOnly:   slices.Contains(q.Filter, "active"),
	})
	if err != nil {
		return Page[*WorkshopCard]{}, err
	}
	return pageItems(all, q, itemRules[*WorkshopCard]{
		search: func(w *WorkshopCard) []string { return []string{w.Name()} },
		filter: keepAll[*WorkshopCard],
		sorts: map[string]func(a, b *WorkshopCard) int{
			"name":             byText(func(w *WorkshopCard) string { return w.Name() }),
			"createdAt":        byValue(func(w *WorkshopCard) int64 { return w.CreatedAt().UnixNano() }),
			"modifiedAt":       byValue(func(w *WorkshopCard) int64 { return w.ModifiedAt().UnixNano() }),
			"participantCount": byValue(func(w *WorkshopCard) int { return len(w.Participants()) }),
		},
	}), nil
}

// Participants is workshop id's participant list, as its card lists them;
// empty for a caller who may not.
func Participants(id uuid.UUID) Lister[WorkshopParticipant] { return participants{id} }

type participants struct{ workshop uuid.UUID }

func (participants) Spec() ListSpec {
	return ListSpec{
		Sorts:       []string{"name", "role", "gamesCount", "createdAt"},
		DefaultSort: "createdAt",
		DefaultDir:  "asc",
	}
}

func (l participants) Query(ctx context.Context, q ListQuery) (Page[WorkshopParticipant], error) {
	w, err := ReadWorkshop(ctx, l.workshop)
	if err != nil {
		return Page[WorkshopParticipant]{}, err
	}
	return pageItems(w.Participants(), q, itemRules[WorkshopParticipant]{
		search: func(p WorkshopParticipant) []string { return []string{p.Name, p.Role} },
		filter: keepAll[WorkshopParticipant],
		sorts: map[string]func(a, b WorkshopParticipant) int{
			"name":       byText(func(p WorkshopParticipant) string { return p.Name }),
			"role":       byText(func(p WorkshopParticipant) string { return p.Role }),
			"gamesCount": byValue(func(p WorkshopParticipant) int { return p.GamesCount }),
			"createdAt":  byValue(func(p WorkshopParticipant) int64 { return p.JoinedAt.UnixNano() }),
		},
	}), nil
}

// ── Keys ───────────────────────────────────────────────────────────────────

// platformFamily names the filters that keep one platform's keys.
const platformFamily = "platform:"

func keepPlatform[T any](platform func(T) string) func(string) func(T) bool {
	return func(name string) func(T) bool {
		want := strings.TrimPrefix(name, platformFamily)
		return func(t T) bool { return platform(t) == want }
	}
}

// OrganisationKeys is the list of the keys given to organisation id.
func OrganisationKeys(id uuid.UUID) Lister[*KeyShareCard] { return organisationKeys{id} }

type organisationKeys struct{ organisation uuid.UUID }

func (organisationKeys) Spec() ListSpec {
	return ListSpec{
		FilterFamilies: []string{platformFamily},
		Sorts:          []string{"name", "owner", "platform"},
		DefaultSort:    "name",
		DefaultDir:     "asc",
	}
}

func (l organisationKeys) Query(ctx context.Context, q ListQuery) (Page[*KeyShareCard], error) {
	all, err := ListKeyShares(ctx, KeyShareFilter{Organisation: &l.organisation})
	if err != nil {
		return Page[*KeyShareCard]{}, err
	}
	return pageItems(all, q, itemRules[*KeyShareCard]{
		search: func(s *KeyShareCard) []string { return []string{s.Key.Name, s.Key.OwnerName} },
		filter: keepPlatform(func(s *KeyShareCard) string { return s.Key.Platform }),
		sorts: map[string]func(a, b *KeyShareCard) int{
			"name":     byText(func(s *KeyShareCard) string { return s.Key.Name }),
			"owner":    byText(func(s *KeyShareCard) string { return s.Key.OwnerName }),
			"platform": byText(func(s *KeyShareCard) string { return s.Key.Platform }),
		},
	}), nil
}

// OwnKey is one of the caller's keys: their own use of it (Self) and every
// share of it, Self included.
type OwnKey struct {
	Self   *KeyShareCard
	Shares []*KeyShareCard
}

// OwnKeys is the list of the caller's keys, oldest first.
var OwnKeys Lister[OwnKey] = ownKeys{}

type ownKeys struct{}

func (ownKeys) Spec() ListSpec {
	return ListSpec{
		FilterFamilies: []string{platformFamily},
		Sorts:          []string{"createdAt", "name", "platform", "status"},
		DefaultSort:    "createdAt",
		DefaultDir:     "asc",
	}
}

func (ownKeys) Query(ctx context.Context, q ListQuery) (Page[OwnKey], error) {
	shares, err := MyKeys(ctx)
	if err != nil {
		return Page[OwnKey]{}, err
	}
	var keys []OwnKey
	for _, s := range shares {
		if !s.Self {
			continue
		}
		key := OwnKey{Self: s}
		for _, other := range shares {
			if other.Key.ID == s.Key.ID {
				key.Shares = append(key.Shares, other)
			}
		}
		keys = append(keys, key)
	}
	return pageItems(keys, q, itemRules[OwnKey]{
		search: func(k OwnKey) []string { return []string{k.Self.Key.Name, k.Self.Key.Platform} },
		filter: keepPlatform(func(k OwnKey) string { return k.Self.Key.Platform }),
		sorts: map[string]func(a, b OwnKey) int{
			"createdAt": byValue(func(k OwnKey) int64 { return k.Self.Key.CreatedAt.UnixNano() }),
			"name":      byText(func(k OwnKey) string { return k.Self.Key.Name }),
			"platform":  byText(func(k OwnKey) string { return k.Self.Key.Platform }),
			"status":    byValue(func(k OwnKey) int { return usageRank(k.Self.Key.LastUsageSuccess) }),
		},
	}), nil
}

// usageRank orders keys by how they did when last used: working, not used
// yet, failed.
func usageRank(success *bool) int {
	switch {
	case success == nil:
		return 1
	case *success:
		return 0
	}
	return 2
}

// ── Verified pages ─────────────────────────────────────────────────────────

// VerifiedPages is the verified pages the caller may see: the admins' allow
// list, waiting first.
func VerifiedPages() Lister[*VerifiedPage] { return verifiedPages{} }

type verifiedPages struct{}

func (verifiedPages) Spec() ListSpec {
	return ListSpec{
		Filters:     []string{"waiting", "approved"},
		Sorts:       []string{"status", "organisation", "createdAt"},
		DefaultSort: "status",
		DefaultDir:  "asc",
	}
}

func (verifiedPages) Query(ctx context.Context, q ListQuery) (Page[*VerifiedPage], error) {
	all, err := ListAllVerifiedPages(ctx)
	if err != nil {
		return Page[*VerifiedPage]{}, err
	}
	approved := func(p *VerifiedPage) bool { return p.ApprovedAt != nil }
	return pageItems(all, q, itemRules[*VerifiedPage]{
		search: func(p *VerifiedPage) []string { return []string{p.OrganisationName, p.PageURL} },
		filter: func(name string) func(*VerifiedPage) bool {
			if name == "approved" {
				return approved
			}
			return func(p *VerifiedPage) bool { return !approved(p) }
		},
		sorts: map[string]func(a, b *VerifiedPage) int{
			"status": byValue(func(p *VerifiedPage) int {
				if approved(p) {
					return 1
				}
				return 0
			}),
			"organisation": byText(func(p *VerifiedPage) string { return p.OrganisationName }),
			"createdAt":    byValue(func(p *VerifiedPage) int64 { return p.CreatedAt.UnixNano() }),
		},
	}), nil
}
