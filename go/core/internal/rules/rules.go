// package: core/internal/rules / the grant table
// type:    data
// job:     who may do what: per role and context, one grant per line
// limits:  vocabulary only (-> grants.go); gates turn it into queries (-> core/gates.go)
package rules

// Rules is who may do what. It is the only place access rights are set:
// whatever no grant names is refused. A role's grants are grouped by the
// context the caller is in (see Context), and within that by entity and
// relation: each block says what the role may do with those.
var Rules = map[Role]Grants{
	// Admin is no god mode: what is not listed here, an admin may not do,
	// e.g. see other people's API keys, spend or share their money, or play
	// their sessions.
	Admin: {
		Always: {
			// running the site; its free-use key is one of the admin's own
			{Read, SiteSettings, Everywhere},
			{Edit, SiteSettings, Everywhere},
			{Manage, Site, Everywhere},
			{Create, OrganisationCard, Everywhere}, // with its operator and address
			{Read, OrganisationCard, Everywhere},
			{List, OrganisationCard, Everywhere},
			{Edit, OrganisationCard, Everywhere},
			{Delete, OrganisationCard, Everywhere},  // for good, with its workshops and guests
			{Suspend, OrganisationCard, Everywhere}, // and lift it; keeps its data, where deleting is for good
			{Read, OrganisationSettings, Everywhere},
			{Edit, OrganisationSettings, Everywhere},
			{Read, OrganisationDetails, Everywhere}, // a head confirms them, consenting
			{Read, OrganisationRecord, Everywhere},
			{Delete, Membership, Everywhere},   // never an organisation's last head
			{List, VerifiedPage, Everywhere},   // the allow list of pages that embed games
			{Manage, VerifiedPage, Everywhere}, // approving a page, or withdrawing the approval

			{Create, WorkshopCard, Everywhere}, // in any organisation
			{Read, WorkshopCard, Everywhere},
			{List, WorkshopCard, Everywhere},
			{Edit, WorkshopCard, Everywhere},
			{Delete, WorkshopCard, Everywhere},
			{Read, WorkshopSettings, Everywhere},
			{Edit, WorkshopSettings, Everywhere},
			{List, WorkshopParticipants, Everywhere},
			{Manage, WorkshopParticipants, Everywhere},
			{Create, Invite, Everywhere},
			{List, Invite, Everywhere},
			{Read, Invite, Everywhere},
			{Delete, Invite, Everywhere},
			{Create, HeadInvite, Everywhere},
			{List, HeadInvite, Everywhere},
			{Read, HeadInvite, Everywhere},
			{Delete, HeadInvite, Everywhere},
			{Edit, Membership, Everywhere}, // any role, admin included; only an individual becomes admin

			{Read, GameCard, Everywhere},
			{List, GameCard, Everywhere},
			{Edit, GameCard, Everywhere},
			{Read, GameContent, Everywhere},
			{Edit, GameContent, Everywhere},
			{Read, GameSettings, Everywhere},
			{Edit, GameSettings, Everywhere},
			{Delete, GameCard, Everywhere},
			{Play, GameCard, Everywhere},    // with their own money: they read every game already
			{Publish, GameCard, Everywhere}, // the admin is liable for what is public, so controls it both ways; v1: only workshop games and the admin's own
			{Copy, GameContent, Everywhere},

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},

			{Read, UserCard, Everywhere},
			{List, UserCard, Everywhere},
			{Edit, UserCard, Everywhere},
			{Read, UserAccount, Everywhere},
			{List, UserAccount, Everywhere},
			{Edit, UserAccount, Everywhere},
			{Read, UserContact, Everywhere},
			{List, UserContact, Everywhere},
			{Edit, UserContact, Everywhere},
			{Delete, UserCard, Everywhere}, // never an organisation's last head
			// no UserSecrets: the admin does not log in as a guest
		},
	},
	Head: {
		Always: {
			{List, OrganisationCard, OfOwnOrganisation},
			{Edit, OrganisationCard, OfOwnOrganisation},
			{Read, OrganisationSettings, OfOwnOrganisation},
			{Edit, OrganisationSettings, OfOwnOrganisation},
			{Read, OrganisationDetails, OfOwnOrganisation},
			{Edit, OrganisationDetails, OfOwnOrganisation}, // consenting to the declaration in its name, or withdrawing it; one head's consent covers it
			{Read, OrganisationRecord, OfOwnOrganisation},
			{Delete, Membership, MemberOfOwnOrganisation}, // leaving too; never the last head
			{Delete, Membership, GuestOfOwnOrganisation},
			{Edit, Membership, MemberOfOwnOrganisation}, // making members head or staff; never the last head away
			{Create, HeadInvite, OfOwnOrganisation},
			{List, HeadInvite, OfOwnOrganisation},
			{Read, HeadInvite, OfOwnOrganisation},
			{Delete, HeadInvite, OfOwnOrganisation},
			{Delete, ApiKeyShare, SharedWithOwnOrganisation}, // removing a key from the organisation; its owner keeps it

			// the organisation's pages that may frame its embedded games
			{Create, VerifiedPage, OfOwnOrganisation},
			{List, VerifiedPage, OfOwnOrganisation},
			{Edit, VerifiedPage, OfOwnOrganisation},
			{Delete, VerifiedPage, OfOwnOrganisation},

			// an organisation is cooperative: its head and staff run all its
			// workshops
			{Create, WorkshopCard, OfOwnOrganisation},
			{Read, WorkshopCard, OfOwnOrganisation},
			{List, WorkshopCard, OfOwnOrganisation},
			{Edit, WorkshopCard, OfOwnOrganisation},
			{Delete, WorkshopCard, OfOwnOrganisation},
			{Read, WorkshopSettings, OfOwnOrganisation},
			{Edit, WorkshopSettings, OfOwnOrganisation},
			{List, WorkshopParticipants, OfOwnOrganisation},
			{Manage, WorkshopParticipants, OfOwnOrganisation}, // taking people out
			{Create, Invite, OfOwnOrganisation},               // staff invite staff, and anyone into a workshop
			{List, Invite, OfOwnOrganisation},
			{Read, Invite, OfOwnOrganisation},
			{Delete, Invite, OfOwnOrganisation}, // whoever made it: the organisation is cooperative
			{Enter, WorkshopCard, OfOwnOrganisation},
			{Leave, WorkshopCard, Current},

			// the organisation's keys, given by their owners: workshops play
			// with them, and they pay for sponsored games and share links
			{List, ApiKeyShare, SharedWithOwnOrganisation},
			{Share, ApiKeyShare, SharedWithOwnOrganisation}, // a sponsorship or a share link drawn from it
			{Use, ApiKeyShare, FreeUseOfOwnOrganisation},
			// what draws from them: the organisation is cooperative, so any of
			// its head and staff stops any of it, whoever made it
			{List, ApiKeyShare, PaidByOwnOrganisation},
			{Manage, ApiKeyShare, PaidByOwnOrganisation},
			{List, ShareLink, PaidByOwnOrganisation},
			{Manage, ShareLink, PaidByOwnOrganisation},

			// the organisation's games: made in its workshops, or by its head or
			// staff while they are; they work on them as their designers do
			{List, GameCard, OfOwnOrganisation},
			{Read, GameCard, OfOwnOrganisation},
			{Read, GameContent, OfOwnOrganisation},
			{Read, GameSettings, OfOwnOrganisation},
			{Edit, GameCard, OfOwnOrganisation},
			{Edit, GameContent, OfOwnOrganisation},
			{Edit, GameSettings, OfOwnOrganisation},
			{Publish, GameCard, OfOwnOrganisation},
			{Play, GameCard, OfOwnOrganisation},
			{Share, GameCard, OfOwnOrganisation}, // embedding included
			{Copy, GameContent, OfOwnOrganisation},
			{Delete, GameCard, OfOwnOrganisationsWorkshop}, // moderating what is made there; a colleague's own game only its designer deletes
			{Edit, SessionRule, Own},                       // tuning how often their session repeats the rule // they run the workshops: in workshop mode or not

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},

			{Read, UserCard, OfOwnOrganisation}, // members, and guests in its workshops
			{Read, UserAccount, OfOwnOrganisation},
			{Read, UserContact, OfOwnOrganisation},
			{Read, UserSecrets, OfOwnOrganisation},      // a guest's login code: helping a guest who lost it
			{Edit, UserSecrets, GuestOfOwnOrganisation}, // a new login code, ending the old one
			{List, UserCard, MemberOfOwnOrganisation},   // listing is narrower: no guests
			{List, UserAccount, MemberOfOwnOrganisation},
			{List, UserContact, MemberOfOwnOrganisation},
			{Edit, UserCard, GuestOfOwnOrganisation},
			{Edit, UserAccount, GuestOfOwnOrganisation},
			{Delete, UserCard, GuestOfOwnOrganisation},
			{Edit, UserContact, Own},
		},
		InWorkshop: {
			{List, GameCard, OfCurrentWorkshop}, // all of them, whatever the workshop's settings
		},
		// a head lifts their organisation's suspension for want of a consent
		// by consenting; an admin's suspension stays (organisation_consent.go)
		Suspended: {
			{Read, OrganisationCard, OfOwnOrganisation},
			{Read, OrganisationSettings, OfOwnOrganisation},
			{Edit, OrganisationSettings, OfOwnOrganisation}, // the restriction, confirmed with the consent
			{Read, OrganisationDetails, OfOwnOrganisation},
			{Edit, OrganisationDetails, OfOwnOrganisation},
			{Read, OrganisationRecord, OfOwnOrganisation},
		},
	},
	Staff: {
		Always: {
			{List, OrganisationCard, OfOwnOrganisation},
			{Edit, OrganisationCard, OfOwnOrganisation},
			{Read, OrganisationSettings, OfOwnOrganisation},
			{Edit, OrganisationSettings, OfOwnOrganisation}, // v1: staff set the restriction too
			{Read, OrganisationDetails, OfOwnOrganisation},
			{Delete, Membership, GuestOfOwnOrganisation},
			{Delete, Membership, Own}, // leaving

			// the organisation's pages that may frame its embedded games
			{Create, VerifiedPage, OfOwnOrganisation},
			{List, VerifiedPage, OfOwnOrganisation},
			{Edit, VerifiedPage, OfOwnOrganisation},
			{Delete, VerifiedPage, OfOwnOrganisation},

			// an organisation is cooperative: its head and staff run all its
			// workshops
			{Create, WorkshopCard, OfOwnOrganisation},
			{Read, WorkshopCard, OfOwnOrganisation},
			{List, WorkshopCard, OfOwnOrganisation},
			{Edit, WorkshopCard, OfOwnOrganisation},
			{Delete, WorkshopCard, OfOwnOrganisation},
			{Read, WorkshopSettings, OfOwnOrganisation},
			{Edit, WorkshopSettings, OfOwnOrganisation},
			{List, WorkshopParticipants, OfOwnOrganisation},
			{Manage, WorkshopParticipants, OfOwnOrganisation}, // taking people out
			{Create, Invite, OfOwnOrganisation},               // staff invite staff, and anyone into a workshop
			{List, Invite, OfOwnOrganisation},
			{Read, Invite, OfOwnOrganisation},
			{Delete, Invite, OfOwnOrganisation}, // whoever made it: the organisation is cooperative
			{Enter, WorkshopCard, OfOwnOrganisation},
			{Leave, WorkshopCard, Current},

			// the organisation's keys, given by their owners: workshops play
			// with them, and they pay for sponsored games and share links
			{List, ApiKeyShare, SharedWithOwnOrganisation},
			{Share, ApiKeyShare, SharedWithOwnOrganisation}, // a sponsorship or a share link drawn from it
			{Use, ApiKeyShare, FreeUseOfOwnOrganisation},
			// what draws from them: the organisation is cooperative, so any of
			// its head and staff stops any of it, whoever made it
			{List, ApiKeyShare, PaidByOwnOrganisation},
			{Manage, ApiKeyShare, PaidByOwnOrganisation},
			{List, ShareLink, PaidByOwnOrganisation},
			{Manage, ShareLink, PaidByOwnOrganisation},

			// the organisation's games: made in its workshops, or by its head or
			// staff while they are; they work on them as their designers do
			{List, GameCard, OfOwnOrganisation},
			{Read, GameCard, OfOwnOrganisation},
			{Read, GameContent, OfOwnOrganisation},
			{Read, GameSettings, OfOwnOrganisation},
			{Edit, GameCard, OfOwnOrganisation},
			{Edit, GameContent, OfOwnOrganisation},
			{Edit, GameSettings, OfOwnOrganisation},
			{Publish, GameCard, OfOwnOrganisation},
			{Play, GameCard, OfOwnOrganisation},
			{Share, GameCard, OfOwnOrganisation}, // embedding included
			{Copy, GameContent, OfOwnOrganisation},
			{Delete, GameCard, OfOwnOrganisationsWorkshop}, // moderating what is made there; a colleague's own game only its designer deletes
			{Edit, SessionRule, Own},                       // tuning how often their session repeats the rule // they run the workshops: in workshop mode or not

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},

			{Read, UserCard, OfOwnOrganisation}, // members, and guests in its workshops
			{Read, UserAccount, OfOwnOrganisation},
			{Read, UserContact, OfOwnOrganisation},
			{Read, UserSecrets, OfOwnOrganisation},      // a guest's login code: helping a guest who lost it
			{Edit, UserSecrets, GuestOfOwnOrganisation}, // a new login code, ending the old one
			{List, UserCard, MemberOfOwnOrganisation},   // listing is narrower: no guests
			{List, UserAccount, MemberOfOwnOrganisation},
			{List, UserContact, MemberOfOwnOrganisation},
			{Edit, UserCard, GuestOfOwnOrganisation},
			{Edit, UserAccount, GuestOfOwnOrganisation},
			{Delete, UserCard, GuestOfOwnOrganisation},
			{Edit, UserContact, Own},
		},
		InWorkshop: {
			{List, GameCard, OfCurrentWorkshop}, // all of them, whatever the workshop's settings
		},
	},
	// Guest has no grants of its own: Everyone's, and SignedIn's in a
	// workshop, are all a guest holds.
	Guest: {},

	// Individual is in no organisation: head and staff are in one, an
	// individual in none, and one an organisation invites becomes head or
	// staff by accepting. Like anyone, an individual takes part in a workshop
	// through its invite, which makes them no part of its organisation: they
	// keep their role, as one watching a film owns no part of the cinema.
	Individual: {
		Always: {
			{Enter, WorkshopCard, ViaInvite},
			{Leave, WorkshopCard, Current},

			{Edit, UserContact, Own},

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},
		},
	},

	// SignedIn is not a security barrier, accounts are free: playing needs
	// the age group and the payer an account brings.
	SignedIn: {
		Always: {
			{Read, SiteRules, Everywhere}, // the fallback below an organisation's
			{Read, UserCard, Own},
			{Edit, UserCard, Own},
			{Read, UserAccount, Own},
			{Edit, UserAccount, Own},
			{Read, UserContact, Own},
			{Read, UserSecrets, Own},
			// changing the own email is the roles' (a guest has none); nobody
			// deletes their own account

			// an account's sessions are its own: nobody else lists, plays or
			// deletes them, the admin included
			{List, Session, Own},
			{Play, Session, Own},
			{Delete, Session, Own},

			{Create, GameCard, Own}, // in a workshop, it belongs to that workshop
			{Play, GameCard, Public},
			{Read, GameSettings, Public}, // v1 shows a public game's settings to every account

			// Copy is reading a game's content and creating a game, nothing
			// more: a grant of its own for convenience. It holds exactly where
			// both do (TestCopyIsReadingAndCreating).
			{Copy, GameContent, Public}, // into a new private game of the caller's
			{Copy, GameContent, Own},
			{Copy, GameContent, ViaShareLink},

			{List, ShareLink, PaidByOwnKey},
			{Manage, ShareLink, PaidByOwnKey}, // the payer stops it; v1 let them only see it

			{List, ShareLink, Own},
			{Manage, ShareLink, Own},

			{List, ShareLink, OfOwnGame},
			{Manage, ShareLink, OfOwnGame},

			// invites addressed to the caller, by account or email
			{List, Invite, SharedWithMe},
			{Read, Invite, SharedWithMe},
			{Use, Invite, SharedWithMe},
			{List, HeadInvite, SharedWithMe},
			{Read, HeadInvite, SharedWithMe},
			{Use, HeadInvite, SharedWithMe},

			// a key is money: whoever bought it owns it, shares it and takes
			// it back; nobody else, the admin included
			{Read, ApiKey, Own}, // the record and a shortened key, never the secret
			{List, ApiKey, Own},
			{Use, ApiKey, Own},
			{Share, ApiKey, Own}, // with their own organisation, or for a sponsored game or a share link
			{Manage, ApiKey, Own},

			{List, ApiKeyShare, PaidByOwnKey},
			{Manage, ApiKeyShare, PaidByOwnKey}, // taking a share back stops what draws from it
		},
		InWorkshop: {
			{Copy, GameContent, OfCurrentWorkshop}, // reading the workshop's games and creating one
		},
	},

	// Everyone is every caller's baseline.
	Everyone: {
		Always: {
			{Read, OrganisationCard, OfOwnOrganisation}, // head's and staff's, theirs
			{Read, OrganisationCard, OfCurrentWorkshop}, // whoever takes part in a workshop, its organisation's; no membership
			{Read, WorkshopCard, Public},                // one with a public page

			{Read, GameCard, Public},
			{Read, GameContent, Public},

			{Read, GameCard, Own},
			{Play, GameCard, Own},
			{Edit, GameCard, Own},
			{Read, GameContent, Own},
			{Edit, GameContent, Own},
			{Read, GameSettings, Own},
			{Edit, GameSettings, Own},
			{Delete, GameCard, Own},
			{Publish, GameCard, Own},

			{Read, GameCard, ViaShareLink},
			{Read, GameContent, ViaShareLink}, // playing includes its insights: prompts, graph
			{Play, GameCard, ViaShareLink},

			{Play, Session, ViaGuestSessionID},

			{Read, Invite, ViaInviteLink}, // a link joins a workshop, without an account too
			{Use, Invite, ViaInviteLink},
		},
		OutsideWorkshop: {
			{List, GameCard, Public},
			{List, GameCard, Own},
		},
		// Anyone in a workshop: a guest in a workshop, always and only in theirs, or
		// head, staff and individuals in workshop mode. A guest through a share
		// link is never in a workshop.
		InWorkshop: {
			{Read, WorkshopCard, Current},
			{List, WorkshopParticipants, Current}, // those in a workshop see each other

			{List, GameCard, OwnOfCurrentWorkshop},
			{Read, GameCard, OfCurrentWorkshop}, // reading is wider than listing
			{Play, GameCard, OfCurrentWorkshop},
			{Read, GameContent, OfCurrentWorkshop},
			{Read, GameSettings, OfCurrentWorkshop},

			{List, ShareLink, OfCurrentWorkshop},

			{Use, ApiKeyShare, SharedWithCurrentWorkshop},
			// no Share on ApiKeyShare here: being in a workshop shares games, never its key
		},
		PublicShown: {
			{List, GameCard, PublicElsewhere}, // own games outside the workshop are not listed
		},
		OthersShown: {
			{List, GameCard, OfCurrentWorkshop},
		},
		// an account signed in but not yet registered completes its own user,
		// and sees it; nothing else of SignedIn holds until then
		Registering: {
			{Create, UserCard, Own},
			{Create, OrganisationCard, Own}, // registering as its head; that it takes an adult is validation in code (organisation_register.go), not this grant
			{Read, UserCard, Own},
			{Read, UserAccount, Own},
			{Read, UserContact, Own},
		},
		// a member of a suspended organisation sees who they are, so the app
		// can tell them; nothing else holds until the suspension ends
		Suspended: {
			{Read, UserCard, Own},
			{Read, UserAccount, Own},
			{Read, UserContact, Own},
		},
		// sharing through the workshop's key, while the workshop allows it;
		// roles with a key of their own share always
		SharingAllowed: {
			{Share, GameCard, Own},
			{Share, GameCard, Public},
			{Share, ApiKeyShare, SharedWithCurrentWorkshop}, // share links paid by the workshop's key
		},
	},
}
