// package: db / database access and repository layer
// type:    data
// job:     seed well-known dev users, institutions, workshops and demo games for local development.
// limits:  does not define SQL queries (-> db/queries) or run in production.
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"cgl/log"
	"cgl/obj"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Well-known UUIDs for dev users (two per role + one guest)
var (
	DevAdmin1UserID      = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	DevAdmin2UserID      = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	DevHead1UserID       = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	DevHead2UserID       = uuid.MustParse("00000000-0000-0000-0000-000000000004")
	DevStaff1UserID      = uuid.MustParse("00000000-0000-0000-0000-000000000005")
	DevStaff2UserID      = uuid.MustParse("00000000-0000-0000-0000-000000000006")
	DevIndividual1UserID = uuid.MustParse("00000000-0000-0000-0000-000000000007")
	DevIndividual2UserID = uuid.MustParse("00000000-0000-0000-0000-000000000008")
	DevGuestUserID       = uuid.MustParse("00000000-0000-0000-0000-000000000009")
	DevInstitutionID     = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	DevWorkshopID        = uuid.MustParse("00000000-0000-0000-0000-000000000011")
)

// DevUserID is the well-known UUID for the dev user (legacy, use DevAdmin1UserID)
var DevUserID = DevAdmin1UserID

// Legacy aliases for backward compatibility
var (
	DevAdminUserID = DevAdmin1UserID
	DevHeadUserID  = DevHead1UserID
	DevStaffUserID = DevStaff1UserID
)

// Preseed ensures required seed data exists in the database
func Preseed(ctx context.Context) {
	log.Debug("running database preseed")

	// Create dev institution first (needed for role assignments)
	preseedDevInstitution(ctx)

	// Create dev workshop (needed for the guest role)
	preseedDevWorkshop(ctx)

	// Create dev users (2 per role + 1 guest)
	preseedDevUsers(ctx)

	// Create mock API key for first admin user
	preseedDevApiKey(ctx, DevAdmin1UserID)

	// Create the demo games, spread across the dev users
	preseedDevGames(ctx)

	log.Debug("database preseed completed")
}

// preseedDevInstitution creates the dev institution if it doesn't exist
func preseedDevInstitution(ctx context.Context) {
	_, err := queries().GetInstitutionByID(ctx, DevInstitutionID)
	if err != nil {
		log.Debug("creating dev institution", "id", DevInstitutionID)
		now := time.Now()
		arg := CreateInstitutionParams{
			ID:         DevInstitutionID,
			CreatedBy:  uuid.NullUUID{},
			CreatedAt:  now,
			ModifiedBy: uuid.NullUUID{},
			ModifiedAt: now,
			Name:       "Dev Organization",
		}
		if _, err := queries().CreateInstitution(ctx, arg); err != nil {
			log.Warn("failed to create dev institution", "error", err)
		}
	}
}

// preseedDevWorkshop creates the dev workshop if it doesn't exist
func preseedDevWorkshop(ctx context.Context) {
	_, err := queries().GetWorkshopByID(ctx, DevWorkshopID)
	if err != nil {
		log.Debug("creating dev workshop", "id", DevWorkshopID)
		now := time.Now()
		arg := CreateWorkshopParams{
			ID:            DevWorkshopID,
			CreatedBy:     uuid.NullUUID{},
			CreatedAt:     now,
			ModifiedBy:    uuid.NullUUID{},
			ModifiedAt:    now,
			Name:          "Dev Workshop",
			InstitutionID: DevInstitutionID,
			Active:        true,
			Public:        false,
		}
		if _, err := queries().CreateWorkshop(ctx, arg); err != nil {
			log.Warn("failed to create dev workshop", "error", err)
		}
	}
}

// devUserSlot is one well-known dev account: a fixed ID with the name and role
// it is meant to carry.
type devUserSlot struct {
	id            uuid.UUID
	name          string
	role          obj.Role
	institutionID *uuid.UUID
}

// devUserSlots lists the dev accounts preseed maintains (2 per role + 1 guest).
func devUserSlots() []devUserSlot {
	return []devUserSlot{
		{DevAdmin1UserID, "admin-1", obj.RoleAdmin, nil},
		{DevAdmin2UserID, "admin-2", obj.RoleAdmin, nil},
		{DevHead1UserID, "head-1", obj.RoleHead, &DevInstitutionID},
		{DevHead2UserID, "head-2", obj.RoleHead, &DevInstitutionID},
		{DevStaff1UserID, "staff-1", obj.RoleStaff, &DevInstitutionID},
		{DevStaff2UserID, "staff-2", obj.RoleStaff, &DevInstitutionID},
		{DevIndividual1UserID, "individual-1", obj.RoleIndividual, nil},
		{DevIndividual2UserID, "individual-2", obj.RoleIndividual, nil},
		{DevGuestUserID, "guest", obj.RoleGuest, &DevInstitutionID},
	}
}

// preseedDevUsers reconciles the well-known dev accounts with devUserSlots.
//
// Misnamed accounts are parked under a temporary name and email before the
// intended ones are applied: the slots have been renumbered before, leaving a
// name wanted by one slot held by the account of another, and name and email
// are both unique columns.
func preseedDevUsers(ctx context.Context) {
	slots := devUserSlots()
	exists := make([]bool, len(slots))
	misnamed := make([]bool, len(slots))

	for i, slot := range slots {
		user, err := GetUserByIDRaw(ctx, slot.id)
		if err != nil {
			continue
		}
		exists[i] = true
		misnamed[i] = user.Name.String != slot.name
		if !misnamed[i] {
			continue
		}
		log.Debug("parking dev user name", "user_id", slot.id, "from", user.Name, "to", slot.name)
		parked := "preseed-parked-" + slot.id.String()
		if err := renameDevUser(ctx, slot.id, parked); err != nil {
			log.Warn("failed to park dev user name", "user_id", slot.id, "from", user.Name, "error", err)
			misnamed[i] = false
		}
	}

	for i, slot := range slots {
		switch {
		case misnamed[i]:
			log.Debug("renaming dev user", "user_id", slot.id, "name", slot.name)
			if err := renameDevUser(ctx, slot.id, slot.name); err != nil {
				log.Warn("failed to rename dev user", "user_id", slot.id, "name", slot.name, "error", err)
				continue
			}
		case !exists[i]:
			log.Debug("creating dev user", "user_id", slot.id, "name", slot.name, "role", slot.role)
			email := devUserEmail(slot.name)
			if _, err := CreateUserWithID(ctx, slot.id, slot.name, &email, ""); err != nil {
				log.Warn("failed to create dev user", "name", slot.name, "error", err)
				continue
			}
		}
		setDevUserRole(ctx, slot)
	}
}

// renameDevUser sets a dev user's name and its matching @dev.local email.
func renameDevUser(ctx context.Context, userID uuid.UUID, name string) error {
	return queries().UpdateUser(ctx, UpdateUserParams{
		ID:    userID,
		Name:  userName(name),
		Email: sql.NullString{String: devUserEmail(name), Valid: true},
	})
}

func devUserEmail(name string) string {
	return name + "@dev.local"
}

// setDevUserRole leaves the dev user holding exactly the slot's role. The roles
// are dropped and recreated rather than compared, because GetUserByID surfaces
// only one of several role rows and a stale second role stays invisible to a
// comparison. Nothing references user_role.id, so deleting is safe.
func setDevUserRole(ctx context.Context, slot devUserSlot) {
	if slot.role == "" {
		return
	}
	if err := queries().DeleteUserRoles(ctx, slot.id); err != nil {
		log.Warn("failed to clear dev user roles", "name", slot.name, "error", err)
		return
	}

	log.Debug("assigning role to dev user", "name", slot.name, "role", slot.role)
	var workshopID uuid.NullUUID
	if slot.role == obj.RoleGuest {
		workshopID = uuid.NullUUID{UUID: DevWorkshopID, Valid: true}
	}
	arg := CreateUserRoleParams{
		UserID:        slot.id,
		Role:          string(slot.role),
		InstitutionID: uuid.NullUUID{UUID: uuidPtrToUUID(slot.institutionID), Valid: slot.institutionID != nil},
		WorkshopID:    workshopID,
	}
	if _, err := queries().CreateUserRole(ctx, arg); err != nil {
		log.Warn("failed to assign role to dev user", "name", slot.name, "role", slot.role, "error", err)
	}
}

// preseedDevApiKey creates a mock API key for the given user if they don't have one
func preseedDevApiKey(ctx context.Context, userID uuid.UUID) {
	user, err := GetUserByID(ctx, userID)
	if err != nil {
		return
	}

	if len(user.ApiKeys) == 0 {
		log.Debug("creating mock API key for dev user", "user_id", userID)
		keyID, err := CreateApiKey(ctx, userID, "Dev Mock Key", "mock", "mock-api-key-for-testing")
		if err != nil {
			log.Warn("failed to create mock API key", "error", err)
			return
		}

		// Set it as the default
		shares, err := GetApiKeySharesByUser(ctx, userID)
		if err != nil {
			log.Warn("failed to get shares", "error", err)
			return
		}
		for _, share := range shares {
			if share.ApiKeyID == *keyID {
				if err := SetUserDefaultApiKeyShare(ctx, userID, &share.ID); err != nil {
					log.Warn("failed to set default API key", "error", err)
				}
				break
			}
		}
	}
}

// devGamesDir holds the demo game files, relative to the checkout root.
var devGamesDir = filepath.Join("testing", "testdata", "games")

// preseedDevGames creates a game from every demo game file, owned by the dev
// users in turn (round-robin over devUserSlots, in file name order); every
// second one is public (devGamePublic). It only
// runs on a fresh instance, while no dev user owns a game yet, so games
// deleted or changed during development stay that way.
func preseedDevGames(ctx context.Context) {
	slots := devUserSlots()
	for _, slot := range slots {
		owned, err := GetGames(ctx, &slot.id, &GetGamesFilters{Filter: "own"})
		if err != nil {
			log.Warn("failed to get games", "user_id", slot.id, "error", err)
			return
		}
		if len(owned) > 0 {
			return
		}
	}

	dir, err := checkoutPath(devGamesDir)
	if err != nil {
		log.Warn("no demo games to seed", "error", err)
		return
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil || len(paths) == 0 {
		log.Warn("no demo games to seed", "dir", dir, "error", err)
		return
	}

	log.Debug("seeding demo games", "count", len(paths), "dir", dir)
	now := time.Now()
	for i, path := range paths {
		owner := slots[i%len(slots)].id
		public := devGamePublic(i)
		history := devGameHistory(filepath.Base(path), public, now)
		if err := preseedDevGame(ctx, owner, path, public, history); err != nil {
			log.Warn("failed to seed demo game", "file", filepath.Base(path), "user_id", owner, "error", err)
		}
	}
}

// devGamePublic says whether the i-th demo game is public: every second one,
// so lists and their public filter have games to show. With an odd number of
// dev users, each of them owns public and private games.
func devGamePublic(i int) bool {
	return i%2 == 0
}

// gameHistory is a demo game's past: when it was made and last changed,
// and how often it was played.
type gameHistory struct {
	createdAt  time.Time
	modifiedAt time.Time
	playCount  int32
}

// devGameHistory makes up a past for a demo game, so lists have something to
// sort by: created within the last year, changed at some point since, and
// played a few times to a few hundred (public games far more than private
// ones). It is random but fixed per file name: every fresh instance looks the
// same.
func devGameHistory(file string, public bool, now time.Time) gameHistory {
	h := fnv.New64a()
	h.Write([]byte(file))
	r := rand.New(rand.NewPCG(h.Sum64(), 0x5eed))

	year := 365 * 24 * time.Hour
	created := now.Add(-time.Duration(r.Float64() * float64(year))).Truncate(time.Second)
	modified := created.Add(time.Duration(r.Float64() * float64(now.Sub(created)))).Truncate(time.Second)

	// A long tail: most games are played a little, a few a lot.
	most := 20.0
	if public {
		most = 500
	}
	plays := int32(math.Exp(r.Float64()*math.Log(most+1))) - 1

	return gameHistory{createdAt: created, modifiedAt: modified, playCount: plays}
}

// preseedDevGame creates a game for owner from the game file at path, with
// the given visibility and past.
func preseedDevGame(ctx context.Context, owner uuid.UUID, path string, public bool, history gameHistory) error {
	file, err := readGameFile(path)
	if err != nil {
		return err
	}
	game := &obj.Game{Name: file.Name, Genre: file.Genre, Public: public}
	if err := CreateGame(ctx, owner, game); err != nil {
		return err
	}
	if err := UpdateGameFromFile(ctx, owner, game.ID, file); err != nil {
		return err
	}
	return queries().SetDevGameHistory(ctx, SetDevGameHistoryParams{
		ID:         game.ID,
		CreatedAt:  history.createdAt,
		ModifiedAt: history.modifiedAt,
		PlayCount:  history.playCount,
	})
}

// readGameFile reads a game file. Like the game file endpoints, it reads the
// YAML through JSON, so the JSON field names apply.
func readGameFile(path string) (obj.GameFile, error) {
	var file obj.GameFile
	blob, err := os.ReadFile(path)
	if err != nil {
		return file, err
	}
	var doc any
	if err := yaml.Unmarshal(blob, &doc); err != nil {
		return file, err
	}
	j, err := json.Marshal(doc)
	if err != nil {
		return file, err
	}
	err = json.Unmarshal(j, &file)
	return file, err
}

// checkoutPath resolves rel against the checkout root: the nearest directory
// up from the working directory that holds schemaf.toml. The dev backend runs
// from the checkout.
func checkoutPath(rel string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "schemaf.toml")); err == nil {
			return filepath.Join(dir, rel), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("schemaf.toml not found above the working directory")
		}
		dir = parent
	}
}
