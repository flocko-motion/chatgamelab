package core

import (
	"context"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/obj"

	"github.com/google/uuid"
)

// What goes when something is deleted, for a caller the deleting function
// has checked. Shares end as taking them back ends them (endShares): their
// share links with their guests, and the shares drawn from them.

// deleteGameRows deletes what hangs on a game, then the game: soft, or for
// good with its deleted creator.
func deleteGameRows(ctx context.Context, q *db.Queries, id uuid.UUID, forGood bool) error {
	if err := q.V2DeleteGameTagsByGameID(ctx, id); err != nil {
		return err
	}
	if err := q.V2DeleteFavoritesByGameID(ctx, id); err != nil {
		return err
	}
	if _, err := q.V2DeleteGameSessionsByGameID(ctx, id); err != nil {
		return err
	}
	// the share links' guests before the links they hang on, the links
	// before the shares drawn for them
	if err := q.V2DeleteGameLinkGuests(ctx, id); err != nil {
		return err
	}
	if err := q.V2DeleteGameSharesByGameID(ctx, id); err != nil {
		return err
	}
	if err := q.V2DeleteApiKeySharesByGameID(ctx, uuid.NullUUID{UUID: id, Valid: true}); err != nil {
		return err
	}
	if forGood {
		if err := q.V2DeleteGameCover(ctx, id); err != nil {
			return err
		}
		return q.V2HardDeleteGame(ctx, id)
	}
	return q.V2SoftDeleteGame(ctx, id)
}

// deleteAccount deletes an account for good: its games, its keys with all
// they pay for, its favourites, roles and invites, then the account and its
// sessions.
func deleteAccount(ctx context.Context, user uuid.UUID) error {
	q := store.Queries()
	games, err := q.V2GamesCreatedBy(ctx, uuid.NullUUID{UUID: user, Valid: true})
	if err != nil {
		return err
	}
	for _, game := range games {
		if err := store.InTx(ctx, func(q *db.Queries) error { return deleteGameRows(ctx, q, game, true) }); err != nil {
			return err
		}
	}
	keys, err := q.V2KeysOf(ctx, user)
	if err != nil {
		return err
	}
	for _, key := range keys {
		rows, err := q.V2SharesOfKey(ctx, key)
		if err != nil {
			return err
		}
		shares := make([]sharedLink, len(rows))
		for i, r := range rows {
			shares[i] = sharedLink{share: r.ID, link: ptrUUID(r.LinkID)}
		}
		if err := endShares(ctx, shares); err != nil {
			return err
		}
		if err := q.V2DeleteKey(ctx, key); err != nil {
			return err
		}
	}
	return store.InTx(ctx, func(q *db.Queries) error {
		id := uuid.NullUUID{UUID: user, Valid: true}
		for _, step := range []func() error{
			func() error { return q.V2ClearDefaultShare(ctx, user) },
			func() error { return q.V2DeleteFavouritesOf(ctx, user) },
			func() error { return q.V2DeleteUserRoles(ctx, user) },
			func() error { return q.V2ClearFreeUseKeyOf(ctx, user) },
			func() error { return q.V2DeleteInvitesFor(ctx, id) },
			func() error { return q.V2ClearInvitesAcceptedBy(ctx, id) },
			func() error { return q.V2ClearOriginalCreator(ctx, id) },
			func() error { return q.V2ClearCoverCreator(ctx, id) },
			func() error { return q.V2DeleteAccount(ctx, user) },
		} {
			if err := step(); err != nil {
				return err
			}
		}
		return nil
	})
}

// deleteWorkshop deletes a workshop: its public page's links end, its
// guests' accounts go (a guest's place is fixed: without it, they have
// none), whoever works in it leaves it, and its games stay with their
// creators.
func deleteWorkshop(ctx context.Context, id uuid.UUID) error {
	if err := endPublicPageLinks(ctx, &id, nil); err != nil {
		return err
	}
	q := store.Queries()
	guests, err := q.V2WorkshopGuests(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		return err
	}
	for _, guest := range guests {
		if err := deleteAccount(ctx, guest); err != nil {
			return err
		}
	}
	return store.InTx(ctx, func(q *db.Queries) error {
		ws := uuid.NullUUID{UUID: id, Valid: true}
		if err := q.V2LeaveDeletedWorkshop(ctx, ws); err != nil {
			return err
		}
		if err := q.V2UnlinkWorkshopGames(ctx, ws); err != nil {
			return err
		}
		return q.V2SoftDeleteWorkshop(ctx, id)
	})
}

// deleteOrganisation deletes an organisation for good: its workshops with
// their guests, every share given to it and share link of it with all they
// pay for, its invites; its head and staff become individuals, in no
// organisation.
func deleteOrganisation(ctx context.Context, id uuid.UUID) error {
	q := store.Queries()
	workshops, err := q.V2OrganisationWorkshops(ctx, id)
	if err != nil {
		return err
	}
	for _, w := range workshops {
		if err := deleteWorkshop(ctx, w); err != nil {
			return err
		}
	}
	rows, err := q.V2OrganisationSharesAndLinks(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		return err
	}
	shares := make([]sharedLink, len(rows))
	for i, r := range rows {
		shares[i] = sharedLink{share: r.ID, link: ptrUUID(r.LinkID)}
	}
	if err := endShares(ctx, shares); err != nil {
		return err
	}
	members, err := q.V2OrganisationMembers(ctx, uuid.NullUUID{UUID: id, Valid: true})
	if err != nil {
		return err
	}
	return store.InTx(ctx, func(q *db.Queries) error {
		org := uuid.NullUUID{UUID: id, Valid: true}
		if err := q.V2DeleteOrganisationInvites(ctx, id); err != nil {
			return err
		}
		if err := q.V2DeleteOrganisationRoles(ctx, org); err != nil {
			return err
		}
		for _, m := range members {
			if _, err := q.V2CreateUserRole(ctx, db.V2CreateUserRoleParams{UserID: m, Role: string(obj.RoleIndividual)}); err != nil {
				return err
			}
		}
		if err := q.V2HardDeleteOrganisationWorkshops(ctx, id); err != nil {
			return err
		}
		return q.V2HardDeleteOrganisation(ctx, id)
	})
}
