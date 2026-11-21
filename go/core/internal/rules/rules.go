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
			{Read, Organisation, Everywhere},
			{List, Organisation, Everywhere},
			{Manage, Organisation, Everywhere},

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
			{Publish, GameCard, Everywhere}, // the admin is liable for what is public, so controls it both ways; v1: only workshop games and the admin's own

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},
		},
	},
	Head: {
		Always: {
			{Manage, Organisation, OfOwnOrganisation},

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
			{Publish, GameCard, OfOwnOrganisation},

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},
		},
		InWorkshop: {
			{List, GameCard, OfCurrentWorkshop}, // all of them, whatever the workshop's settings
			{Read, GameCard, OfCurrentWorkshop},
			{Read, GameContent, OfCurrentWorkshop},
			{Read, GameSettings, OfCurrentWorkshop},
		},
	},
	Staff: {
		Always: {
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
			{Publish, GameCard, OfOwnOrganisation},

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},
		},
		InWorkshop: {
			{List, GameCard, OfCurrentWorkshop}, // all of them, whatever the workshop's settings
			{Read, GameCard, OfCurrentWorkshop},
			{Read, GameContent, OfCurrentWorkshop},
			{Read, GameSettings, OfCurrentWorkshop},
		},
	},
	// Participant is always in its workshop.
	Participant: {
		InWorkshop: {
			{List, GameCard, OwnOfCurrentWorkshop},
			{Read, GameCard, OfCurrentWorkshop}, // reading is wider than listing
			{Play, GameCard, OfCurrentWorkshop},
			{Read, GameContent, OfCurrentWorkshop},
			{Read, GameSettings, OfCurrentWorkshop},

			{List, ShareLink, OfCurrentWorkshop},

			{Use, ApiKeyShare, SharedWithCurrentWorkshop},
			// no Share on ApiKeyShare: a participant shares games, never the key
		},
		ParticipantsShown: {
			{List, GameCard, OfCurrentWorkshop},
		},
		// a participant has no key of their own: they share only through the
		// workshop's key, while the workshop allows it
		SharingAllowed: {
			{Share, GameCard, Own},
			{Share, GameCard, Public},
		},
	},
	// Individual has no organisation; in workshop mode, it is in a workshop
	// like a participant.
	Individual: {
		Always: {
			{Enter, Workshop, Active},

			{Share, GameCard, Own}, // if sharing is possible at all: who pays is the key share's (#451)
			{Share, GameCard, Public},
		},
		InWorkshop: {
			{Read, Workshop, Current},

			{List, GameCard, OwnOfCurrentWorkshop},
			{Read, GameCard, OfCurrentWorkshop}, // reading is wider than listing
			{Play, GameCard, OfCurrentWorkshop},
			{Read, GameContent, OfCurrentWorkshop},
			{Read, GameSettings, OfCurrentWorkshop},
		},
		ParticipantsShown: {
			{List, GameCard, OfCurrentWorkshop},
		},
	},

	// SignedIn is not a security barrier, accounts are free: playing needs
	// the age group and the payer an account brings.
	SignedIn: {
		Always: {
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

	// Everyone is the baseline: a guest has exactly this.
	Everyone: {
		Always: {
			{Read, GameCard, Public},
			{Read, GameContent, Public},

			{Read, GameCard, Own},
			{Play, GameCard, Own},
			{Edit, GameCard, Own},
			{Read, GameContent, Own},
			{Edit, GameContent, Own},
			{Read, GameSettings, Own},
			{Edit, GameSettings, Own},
			{Publish, GameCard, Own},

			{Read, GameCard, ViaShareLink}, // the card only: v1 hides the content here
			{Play, GameCard, ViaShareLink},

			{Play, Session, ViaGuestSessionID},
		},
		OutsideWorkshop: {
			{List, GameCard, Public},
			{List, GameCard, Own},
		},
		PublicShown: {
			{List, GameCard, PublicElsewhere}, // own games outside the workshop are not listed
		},
	},
}
