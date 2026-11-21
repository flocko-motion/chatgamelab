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
			{Delete, Membership, Everywhere}, // never an organisation's last head

			{Read, Workshop, Everywhere},
			{List, Workshop, Everywhere},
			{Manage, Workshop, Everywhere},

			{Read, GameCard, Everywhere},
			{List, GameCard, Everywhere},
			{Edit, GameCard, Everywhere},
			{Read, GameContent, Everywhere},
			{Edit, GameContent, Everywhere},
			{Read, GameSettings, Everywhere},
			{Edit, GameSettings, Everywhere},
			{Delete, GameCard, Everywhere},
			{Publish, GameCard, Everywhere}, // the admin is liable for what is public, so controls it both ways; v1: only workshop games and the admin's own

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

			{Manage, Workshop, OfOwnOrganisation},
			{Enter, Workshop, OfOwnOrganisation},

			{List, ApiKeyShare, SharedWithOwnOrganisation},
			{Use, ApiKeyShare, SharedWithOwnOrganisation},
			{Manage, ApiKeyShare, SharedWithOwnOrganisation},

			{Share, ApiKeyShare, Held}, // where to: sharing.go

			{Read, GameCard, OfOwnOrganisation}, // v1: any game of the organisation's workshops
			{Read, GameContent, OfOwnOrganisation},
			{Read, GameSettings, OfOwnOrganisation},
			{Edit, GameCard, OfOwnOrganisation}, // v1: any game of the organisation's workshops
			{Edit, GameContent, OfOwnOrganisation},
			{Edit, GameSettings, OfOwnOrganisation},
			{Delete, GameCard, OfOwnOrganisation},
			{Publish, GameCard, OfOwnOrganisation},

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},

			{Read, UserCard, OfOwnOrganisation}, // members, and guests in its workshops
			{Read, UserAccount, OfOwnOrganisation},
			{Read, UserContact, OfOwnOrganisation},
			{Read, UserSecrets, OfOwnOrganisation},    // a guest's login code: helping a guest who lost it
			{List, UserCard, MemberOfOwnOrganisation}, // listing is narrower: no guests
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

			{Manage, Workshop, Assigned},
			{Enter, Workshop, OfOwnOrganisation},

			{List, ApiKeyShare, SharedWithOwnOrganisation},
			{Use, ApiKeyShare, SharedWithOwnOrganisation},
			{Manage, ApiKeyShare, SharedWithOwnOrganisation},

			{Share, ApiKeyShare, Held}, // where to: sharing.go

			{Read, GameCard, OfOwnOrganisation}, // v1: any game of the organisation's workshops
			{Read, GameContent, OfOwnOrganisation},
			{Read, GameSettings, OfOwnOrganisation},
			{Edit, GameCard, OfOwnOrganisation}, // v1: any game of the organisation's workshops
			{Edit, GameContent, OfOwnOrganisation},
			{Edit, GameSettings, OfOwnOrganisation},
			{Delete, GameCard, OfOwnOrganisation},
			{Publish, GameCard, OfOwnOrganisation},

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},

			{Read, UserCard, OfOwnOrganisation}, // members, and guests in its workshops
			{Read, UserAccount, OfOwnOrganisation},
			{Read, UserContact, OfOwnOrganisation},
			{Read, UserSecrets, OfOwnOrganisation},    // a guest's login code: helping a guest who lost it
			{List, UserCard, MemberOfOwnOrganisation}, // listing is narrower: no guests
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

	// Individual may belong to an organisation or to none; in workshop mode,
	// it is in a workshop like anyone (Everyone, InWorkshop).
	Individual: {
		Always: {
			{Enter, Workshop, Active},

			{List, UserCard, MemberOfOwnOrganisation}, // an individual of an organisation sees who else is in it
			{Delete, Membership, Own},                 // leaving it

			{Edit, UserContact, Own},

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},
		},
	},

	// SignedIn is not a security barrier, accounts are free: playing needs
	// the age group and the payer an account brings.
	SignedIn: {
		Always: {
			{Read, UserCard, Own},
			{Edit, UserCard, Own},
			{Read, UserAccount, Own},
			{Edit, UserAccount, Own},
			{Read, UserContact, Own},
			{Read, UserSecrets, Own},
			// changing the own email is the roles' (a guest has none); nobody
			// deletes their own account

			{Create, GameCard, Own}, // in a workshop, it belongs to that workshop
			{Play, GameCard, Public},
			{Read, GameSettings, Public}, // v1 shows a public game's settings to every account

			{Copy, GameContent, Public}, // into a new private game of the caller's
			{Copy, GameContent, Own},

			{List, ShareLink, PaidByOwnKey}, // v1: the payer may not revoke; questionable

			{List, ShareLink, Own},
			{Manage, ShareLink, Own},

			{List, ShareLink, OfOwnGame},
			{Manage, ShareLink, OfOwnGame},

			{Read, ApiKey, Own}, // the record and a shortened key, never the secret
			{List, ApiKey, Own},
			{Use, ApiKey, Own},
			{Share, ApiKey, Own},
			{Manage, ApiKey, Own},

			{List, ApiKeyShare, PaidByOwnKey},
			{Manage, ApiKeyShare, PaidByOwnKey},

			{List, ApiKeyShare, SharedWithMe},
			{Use, ApiKeyShare, SharedWithMe},
			{Manage, ApiKeyShare, SharedWithMe}, // v1: the recipient may drop it
		},
	},

	// Everyone is every caller's baseline.
	Everyone: {
		Always: {
			{Read, OrganisationCard, OfOwnOrganisation}, // a member's, or a guest's through their workshop

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

			{Read, GameCard, ViaShareLink}, // the card only: v1 hides the content here
			{Play, GameCard, ViaShareLink},

			{Play, Session, ViaGuestSessionID},
		},
		OutsideWorkshop: {
			{List, GameCard, Public},
			{List, GameCard, Own},
		},
		// Anyone in a workshop: a guest in a workshop, always and only in theirs, or
		// head, staff and individuals in workshop mode. A guest through a share
		// link is never in a workshop.
		InWorkshop: {
			{Read, Workshop, Current},

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
		},
	},
}
