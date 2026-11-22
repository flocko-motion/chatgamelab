// Package rules is the secure core's access rules: who may do what. It holds
// only vocabulary and the grant table, no code. DRAFT.
package rules

// Role is what a caller is in the system. A role is nothing but a list of
// grants (see rules.go).
type Role string

// The role says where someone is (spec, Roles). Head and staff are in an
// organisation; an individual is in none. An organisation invites an
// individual as head or staff, and by accepting they take that role;
// whoever leaves an organisation, or is removed from it, is an individual
// again. Workshops belong to an organisation: its head and staff run them,
// guests take part in them, an individual takes part in none.
const (
	// Admin runs the site.
	Admin Role = "admin"
	// Head leads an organisation; Staff runs its workshops.
	Head  Role = "head"
	Staff Role = "staff"
	// Individual uses the platform in no organisation and no workshop.
	Individual Role = "individual"

	// Guest plays a shared game through a share link, or takes part in a
	// workshop through its invite link under an animal name. A guest through
	// a share link never signs in and is never in a workshop; a guest in a
	// workshop signs in (so SignedIn holds too) and is always in its
	// workshop: no grant lets a guest enter or leave one. A caller nobody
	// signed in is a guest too.
	Guest Role = "guest"

	// Everyone is every caller, signed in or not, on top of their role.
	Everyone Role = "everyone"

	// SignedIn is every caller with an account that completed registration,
	// on top of Everyone. While registering, an account holds only what
	// Everyone may in the Registering context.
	SignedIn Role = "signed in"
)

// Action is what is done with an entity.
//
// Read opens one entity whose address the caller has; List finds entities.
// Whoever may list something may also read it, not the other way round.
// Listing a secret, such as share links, hands out the secrets themselves.
type Action string

const (
	Read Action = "read"
	List Action = "list"
	Play Action = "play"
	// Edit changes an entity. Deleting it is Delete: whoever may change an
	// entity may not always remove it (everyone edits their own account,
	// nobody deletes it).
	Edit   Action = "edit"
	Delete Action = "delete"
	Manage Action = "manage"
	// Create makes a new entity. Own: it is the caller's from the start, and
	// its rights are the role's Own grants. OfOwnOrganisation: it belongs to
	// the caller's organisation, as a workshop does. Everywhere: the caller
	// makes it for others, as an admin makes an organisation.
	Create Action = "create"
	// Copy makes a new entity of the caller's from this one.
	Copy Action = "copy"
	// Publish makes an entity public: from then on, everyone finds it.
	// Making it private again only narrows who sees it, so it is an Edit.
	Publish Action = "publish"

	// Use spends from a key for the caller's own plays; Share passes it on
	// (a key share, a share link, a workshop's key, sponsoring a game).
	Use   Action = "use"
	Share Action = "share"

	// Enter puts the caller into a workshop: workshop mode. Leave takes them
	// out again. A guest never leaves: their workshop is their place on the
	// platform, which the model relies on.
	Enter Action = "enter"
	Leave Action = "leave"

	// Suspend stops an entity's use, and lifts that again; its data stays.
	// A suspended organisation's members are refused (see Suspended).
	Suspend Action = "suspend"
)

// Entity is a kind of thing in the system, named in the singular. Where
// sections of a thing differ in who may see them, the thing is split and each
// section is an entity of its own (GameCard, GameContent, GameSettings); the
// whole thing is then no entity. Otherwise a thing is one entity.
type Entity string

const (
	// GameCard is a game's card: name, description, image, genre, tags, play
	// count. It stands for the game in lists and is what is played; it
	// reveals neither content nor settings.
	GameCard Entity = "game card"
	// GameContent is how a game works: prompts, style, status fields, first
	// scene. Open in the spirit of ChatGameLab.
	GameContent Entity = "game content"
	// GameSettings is who is behind a game: its workshop and organisation,
	// the key that pays, its creator.
	GameSettings Entity = "game settings"

	// Session is a game being played: its story, state and insights.
	Session Entity = "session"
	// SessionRule is how a session applies the youth-protection rule: how
	// often each of its blocks repeats it.
	SessionRule Entity = "session rule"
	// ShareLink is a voucher: someone's money, for plays of one game, for
	// whoever holds the link. It belongs to neither the game nor the key; the
	// payer is the weightier side, so whoever pays for it may see it.
	ShareLink Entity = "share link"
	// ApiKey is a key at an AI provider. Its secret is write-only: no grant
	// reads it, not even the owner's.
	ApiKey Entity = "api key"
	// ApiKeyShare is the right to spend from a key, given to a user, a
	// workshop or an organisation. Its terms (tiers, features, passing on)
	// are data on the share, not rules here.
	ApiKeyShare Entity = "api key share"

	// UserCard is who a user is: their id and public profile name, the name
	// games show as their creator's.
	UserCard Entity = "user card"
	// UserAccount is how a user's account is set up: language, age group,
	// AI quality tier, default key, role (with its organisation and
	// workshop), timestamps and statistics.
	UserAccount Entity = "user account"
	// UserContact is how to reach a user: their email. Seen like the account,
	// changed by its owner; a guest has none.
	UserContact Entity = "user contact"
	// UserSecrets is how a user logs in without signing up: a guest's login
	// code. Whoever holds it is that guest.
	UserSecrets Entity = "user secrets"

	// WorkshopCard is what a workshop is to those in it: its name and
	// organisation, whether it runs or is paused, what it shows and lets its
	// participants do, its AI tier, and whether it has a public page and at
	// which address.
	WorkshopCard Entity = "workshop card"
	// WorkshopSettings is how a workshop is set up behind that: its
	// youth-protection restriction, the key share it plays with, and its
	// public page's text and links.
	WorkshopSettings Entity = "workshop settings"
	// WorkshopParticipants is who is in a workshop: its guests, and the head,
	// staff and individuals working in it. Managing it takes someone out of
	// the workshop; taking part makes no one part of its organisation.
	WorkshopParticipants Entity = "workshop participants"
	// Invite brings someone into an organisation as staff (as head: a
	// HeadInvite), or into a workshop: for an account or an email, or as a link anyone may
	// follow. Listing it hands out its link. Using it answers it: accepting
	// or declining.
	Invite Entity = "invite"
	// HeadInvite is an invite to become an organisation's head: kept apart,
	// as fewer may make, see or revoke it.
	HeadInvite Entity = "head invite"

	// OrganisationCard is who an organisation is: its id and name, what its
	// members and the guests in its workshops see of it.
	OrganisationCard Entity = "organisation card"
	// OrganisationSettings is how an organisation runs: its youth-protection
	// restriction and the key share its free use spends from.
	OrganisationSettings Entity = "organisation settings"
	// OrganisationDetails is who stands behind an organisation: its operator
	// (Träger) and address, which a head confirms with the organisation's
	// consent to the declaration. Consenting and withdrawing are a head's
	// edits of it.
	OrganisationDetails Entity = "organisation details"
	// OrganisationRecord is an organisation's standing: its consent status,
	// how it registered itself, and an admin's suspension with its reason.
	OrganisationRecord Entity = "organisation record"
	// Membership is a user's role in an organisation. Deleting it removes
	// the user from the organisation; they keep their account, as an
	// individual.
	Membership Entity = "membership"
	// Site is the running platform itself, such as restarting the server.
	Site Entity = "site"
	// SiteSettings are the site's settings: its rules, tiers and free-use key.
	SiteSettings Entity = "site settings"
	// SiteRules are the site's youth-protection rules per age group, the
	// fallback below an organisation's.
	SiteRules Entity = "site rules"
)

// Relation says which entities a grant covers, seen from the caller. A
// relation means the same for every entity it is used with. The caller's
// situation, such as being in a workshop or what it shows, is the grant's
// context (Grants), never part of a relation.
type Relation string

const (
	Everywhere Relation = "everywhere"
	Public     Relation = "public"
	Active     Relation = "active"  // switched on, e.g. a workshop that is running
	Current    Relation = "current" // the one the caller is in, e.g. their current workshop
	Own        Relation = "own"     // created by the caller

	OfCurrentWorkshop Relation = "of current workshop" // the workshop the caller is in
	OfOwnOrganisation Relation = "of own organisation" // belongs to the caller's organisation, e.g. a game of one of its workshops, a member, a guest in one of its workshops

	MemberOfOwnOrganisation Relation = "member of own organisation" // head or staff of the caller's organisation; an individual is in none
	GuestOfOwnOrganisation  Relation = "guest of own organisation"  // a guest in one of the caller's organisation's workshops
	OfOwnGame               Relation = "of own game"                // belongs to a game the caller created
	PaidByOwnKey            Relation = "paid by own key"            // spends from an API key the caller owns
	PaidByOwnOrganisation   Relation = "paid by own organisation"   // spends from a key given to the caller's organisation

	OwnOfCurrentWorkshop Relation = "own, of the current workshop" // created by the caller, in the workshop they are in
	PublicElsewhere      Relation = "public, elsewhere"            // public, not of the workshop the caller is in

	SharedWithMe              Relation = "shared with me"               // given to the caller
	SharedWithCurrentWorkshop Relation = "shared with current workshop" // given to the workshop the caller is in
	SharedWithOwnOrganisation Relation = "shared with own organisation" // given to the caller's organisation
	FreeUseOfOwnOrganisation  Relation = "free use of own organisation" // the key the caller's organisation lets its members play with

	// Via: the caller holds the secret that opens this one entity. Like a
	// wallet key, it grants access but can never be listed.
	ViaShareLink      Relation = "via share link"
	ViaInviteLink     Relation = "via invite link" // an invite link: whoever has it may follow it
	ViaInvite         Relation = "via invite"      // through an invite to it the caller accepts, a link or one addressed to them
	ViaGuestSessionID Relation = "via guest session id"
)

// Context is a situation the caller is in. A grant holds in one context; a
// caller can be in several at once, and holds the grants of all of them.
type Context string

const (
	Always          Context = "always"
	InWorkshop      Context = "in a workshop"      // the caller is in a workshop
	OutsideWorkshop Context = "outside a workshop" // the caller is in none
	PublicShown     Context = "public shown"       // the caller's workshop shows public games
	OthersShown     Context = "others shown"       // the caller's workshop shows the games of the others in it
	SharingAllowed  Context = "sharing allowed"    // the caller's workshop lets those in it share games through its key
	Registering     Context = "registering"        // the caller signed in, but has not completed registration
	// Suspended: the caller's organisation is suspended, by an admin or for
	// want of a consent to the current declaration. It is the caller's only
	// context, so of all their grants only Suspended ones hold.
	Suspended Context = "organisation suspended"
)

// Grants are a role's grants by context.
type Grants map[Context][]Grant

// For are the grants that hold in the given contexts.
func (g Grants) For(contexts []Context) []Grant {
	var grants []Grant
	for _, c := range contexts {
		grants = append(grants, g[c]...)
	}
	return grants
}

// Grant gives a role an action on the entities of a relation: one line of
// the rules.
type Grant struct {
	Action   Action
	Entity   Entity
	Relation Relation
}
