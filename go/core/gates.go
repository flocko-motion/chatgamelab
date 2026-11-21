package core

// The gates: which grants of the table (core/internal/rules) open which
// fragment of a query (go/db/queries/v2_*.sql). Each parameter is one line of
// the table; a maintainer checks table, gates and SQL against each other here.
// gates_test.go fails when the table grants a relation no gate opens.

import (
	"context"

	"cgl/core/internal/rules"
	"cgl/db"

	"github.com/google/uuid"
)

// listGameCardsGates opens V2GameCards for a list: the rows by {List, GameCard}.
func listGameCardsGates(ctx context.Context) db.V2GameCardsParams {
	p := gameCardGates(ctx)
	p.CardPublic = granted(ctx, rules.List, rules.GameCard, rules.Public)
	p.CardOwn = granted(ctx, rules.List, rules.GameCard, rules.Own)
	p.CardPublicElsewhere = granted(ctx, rules.List, rules.GameCard, rules.PublicElsewhere)
	p.CardOwnOfWorkshop = granted(ctx, rules.List, rules.GameCard, rules.OwnOfCurrentWorkshop)
	p.CardWorkshop = granted(ctx, rules.List, rules.GameCard, rules.OfCurrentWorkshop)
	p.CardEverywhere = granted(ctx, rules.List, rules.GameCard, rules.Everywhere)
	return p
}

// readGameCardGates opens V2GameCards for one game: the row by {Read, GameCard}.
func readGameCardGates(ctx context.Context, id uuid.UUID) db.V2GameCardsParams {
	p := gameCardGates(ctx)
	p.ID = uuid.NullUUID{UUID: id, Valid: true}
	p.CardPublic = granted(ctx, rules.Read, rules.GameCard, rules.Public)
	p.CardOwn = granted(ctx, rules.Read, rules.GameCard, rules.Own)
	p.CardWorkshop = granted(ctx, rules.Read, rules.GameCard, rules.OfCurrentWorkshop)
	p.CardOrganisation = granted(ctx, rules.Read, rules.GameCard, rules.OfOwnOrganisation)
	p.CardViaShare = granted(ctx, rules.Read, rules.GameCard, rules.ViaShareLink)
	p.CardEverywhere = granted(ctx, rules.Read, rules.GameCard, rules.Everywhere)
	return p
}

// gameCardGates opens what list and read share: the sections by
// {Read, GameContent} and {Read, GameSettings}, the actions by {Edit, <section>},
// {Copy, GameContent}, {Share, GameCard} and {Publish, GameCard}.
func gameCardGates(ctx context.Context) db.V2GameCardsParams {
	return db.V2GameCardsParams{
		Caller:          account(ctx),
		CurrentWorkshop: currentWorkshop(ctx),
		OwnOrganisation: ownOrganisation(ctx),

		ContentPublic:       granted(ctx, rules.Read, rules.GameContent, rules.Public),
		ContentOwn:          granted(ctx, rules.Read, rules.GameContent, rules.Own),
		ContentWorkshop:     granted(ctx, rules.Read, rules.GameContent, rules.OfCurrentWorkshop),
		ContentOrganisation: granted(ctx, rules.Read, rules.GameContent, rules.OfOwnOrganisation),
		ContentEverywhere:   granted(ctx, rules.Read, rules.GameContent, rules.Everywhere),

		SettingsPublic:       granted(ctx, rules.Read, rules.GameSettings, rules.Public),
		SettingsOwn:          granted(ctx, rules.Read, rules.GameSettings, rules.Own),
		SettingsWorkshop:     granted(ctx, rules.Read, rules.GameSettings, rules.OfCurrentWorkshop),
		SettingsOrganisation: granted(ctx, rules.Read, rules.GameSettings, rules.OfOwnOrganisation),
		SettingsEverywhere:   granted(ctx, rules.Read, rules.GameSettings, rules.Everywhere),

		EditCardOwn:              granted(ctx, rules.Edit, rules.GameCard, rules.Own),
		EditCardOrganisation:     granted(ctx, rules.Edit, rules.GameCard, rules.OfOwnOrganisation),
		EditCardEverywhere:       granted(ctx, rules.Edit, rules.GameCard, rules.Everywhere),
		EditContentOwn:           granted(ctx, rules.Edit, rules.GameContent, rules.Own),
		EditContentOrganisation:  granted(ctx, rules.Edit, rules.GameContent, rules.OfOwnOrganisation),
		EditContentEverywhere:    granted(ctx, rules.Edit, rules.GameContent, rules.Everywhere),
		EditSettingsOwn:          granted(ctx, rules.Edit, rules.GameSettings, rules.Own),
		EditSettingsOrganisation: granted(ctx, rules.Edit, rules.GameSettings, rules.OfOwnOrganisation),
		EditSettingsEverywhere:   granted(ctx, rules.Edit, rules.GameSettings, rules.Everywhere),

		CopyPublic: granted(ctx, rules.Copy, rules.GameContent, rules.Public),
		CopyOwn:    granted(ctx, rules.Copy, rules.GameContent, rules.Own),

		SharePublic: granted(ctx, rules.Share, rules.GameCard, rules.Public),
		ShareOwn:    granted(ctx, rules.Share, rules.GameCard, rules.Own),

		PublishOwn:          granted(ctx, rules.Publish, rules.GameCard, rules.Own),
		PublishOrganisation: granted(ctx, rules.Publish, rules.GameCard, rules.OfOwnOrganisation),
		PublishEverywhere:   granted(ctx, rules.Publish, rules.GameCard, rules.Everywhere),
	}
}
