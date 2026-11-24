package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cgl/core/internal/store"
	"cgl/db"
	"cgl/functional"
	"cgl/legal"
	"cgl/log"
	"cgl/obj"

	"engine"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// The dev preseed (dev mode only, a system job of Bootstrap): a dev
// organisation and workshop, two dev accounts per role and a guest, a mock
// key, the demo games and the site's rules. It reconciles on every start and
// leaves what development changed.

// The dev accounts' fixed ids; the dev sign-in offers them.
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

// preseed runs the dev preseed.
func preseed(ctx context.Context) {
	log.Debug("running database preseed")
	q := store.Queries()
	now := time.Now()
	if _, err := q.V2GetInstitutionByID(ctx, DevInstitutionID); err != nil {
		if _, err := q.V2SeedOrganisation(ctx, db.V2SeedOrganisationParams{ID: DevInstitutionID, Name: "Dev Organization"}); err != nil {
			log.Warn("failed to create dev organisation", "error", err)
		}
	}
	if _, err := q.V2GetWorkshopByID(ctx, DevWorkshopID); err != nil {
		if _, err := q.V2SeedWorkshop(ctx, db.V2SeedWorkshopParams{
			ID: DevWorkshopID, CreatedAt: now, ModifiedAt: now, Name: "Dev Workshop", InstitutionID: DevInstitutionID, Active: true,
		}); err != nil {
			log.Warn("failed to create dev workshop", "error", err)
		}
	}
	preseedDevUsers(ctx)
	preseedDevConsent(ctx)
	preseedDevKey(ctx, DevAdmin1UserID)
	preseedDevPlatformKeys(ctx, DevAdmin1UserID)
	preseedDevGames(ctx)
	preseedDevCovers(ctx)
	log.Debug("database preseed completed")
}

// devUserSlot is one dev account: a fixed id, its name and role, and its
// organisation (head and staff only: an individual is in none, a guest is in
// the dev workshop).
type devUserSlot struct {
	id           uuid.UUID
	name         string
	role         obj.Role
	organisation *uuid.UUID
}

// devUserSlots are the dev accounts: two per role and a guest.
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
		{DevGuestUserID, "guest", obj.RoleGuest, nil},
	}
}

// preseedDevUsers reconciles the dev accounts with devUserSlots. Misnamed
// accounts are parked under a temporary name first: slots were renumbered
// before, and name and email are unique.
func preseedDevUsers(ctx context.Context) {
	q := store.Queries()
	slots := devUserSlots()
	exists := make([]bool, len(slots))
	misnamed := make([]bool, len(slots))
	for i, slot := range slots {
		u, err := q.V2GetUserByID(ctx, slot.id)
		if err != nil {
			continue
		}
		exists[i] = true
		if misnamed[i] = u.Name.String != slot.name; misnamed[i] {
			if err := renameDevUser(ctx, slot.id, "preseed-parked-"+slot.id.String()); err != nil {
				log.Warn("failed to park dev user name", "user_id", slot.id, "error", err)
				misnamed[i] = false
			}
		}
	}
	for i, slot := range slots {
		switch {
		case misnamed[i]:
			if err := renameDevUser(ctx, slot.id, slot.name); err != nil {
				log.Warn("failed to rename dev user", "user_id", slot.id, "error", err)
				continue
			}
		case !exists[i]:
			email := slot.name + "@dev.local"
			if _, err := q.V2SeedUser(ctx, db.V2SeedUserParams{
				ID: slot.id, Name: sql.NullString{String: slot.name, Valid: true}, Email: sql.NullString{String: email, Valid: true},
			}); err != nil && !errors.Is(err, sql.ErrNoRows) {
				log.Warn("failed to create dev user", "name", slot.name, "error", err)
				continue
			}
		}
		setDevUserRole(ctx, slot)
	}
}

// preseedDevConsent keeps the dev organisation's consent to the current
// declaration, given by head-1, so it is not suspended: also after the
// declaration changes.
func preseedDevConsent(ctx context.Context) {
	q := store.Queries()
	declaration := legal.CurrentDeclaration()
	latest, err := q.V2LatestConsent(ctx, DevInstitutionID)
	if err != nil {
		log.Warn("failed to read the dev organisation's consent", "error", err)
		return
	}
	if len(latest) == 1 && latest[0].Kind == "consent" &&
		latest[0].DeclarationVersion == declaration.Version && latest[0].DeclarationHash == declaration.Hash {
		return
	}
	if _, err := recordConsent(ctx, q, DevInstitutionID, DevHead1UserID, legal.Consent{
		PersonName: "head-1", PersonFunction: "Head", Email: "head-1@dev.local",
		Organisation: "Dev Organization", Operator: "Dev Operator",
		Street: "Devstraße 1", PostalCode: "00000", City: "Devstadt",
	}); err != nil {
		log.Warn("failed to record the dev organisation's consent", "error", err)
	}
}

// renameDevUser sets a dev account's name and its matching @dev.local email.
func renameDevUser(ctx context.Context, user uuid.UUID, name string) error {
	return store.Queries().V2SeedRenameUser(ctx, db.V2SeedRenameUserParams{
		ID: user, Name: sql.NullString{String: name, Valid: true}, Email: sql.NullString{String: name + "@dev.local", Valid: true},
	})
}

// setDevUserRole leaves the dev account holding exactly its slot's role.
func setDevUserRole(ctx context.Context, slot devUserSlot) {
	err := store.InTx(ctx, func(q *db.Queries) error {
		if err := q.V2DeleteUserRoles(ctx, slot.id); err != nil {
			return err
		}
		p := db.V2CreateUserRoleParams{UserID: slot.id, Role: string(slot.role), InstitutionID: nullUUID(slot.organisation)}
		if slot.role == obj.RoleGuest {
			p.WorkshopID = uuid.NullUUID{UUID: DevWorkshopID, Valid: true}
		}
		_, err := q.V2CreateUserRole(ctx, p)
		return err
	})
	if err != nil {
		log.Warn("failed to set dev user role", "name", slot.name, "role", slot.role, "error", err)
	}
}

// preseedDevKey gives a dev account a mock key, its default, if it has none.
func preseedDevKey(ctx context.Context, user uuid.UUID) {
	err := store.InTx(ctx, func(q *db.Queries) error {
		count, err := q.V2CountOwnKeys(ctx, user)
		if err != nil || count > 0 {
			return err
		}
		id := uuid.New()
		if err := q.V2CreateKey(ctx, db.V2CreateKeyParams{
			ID: id, Owner: user, Name: "Dev Mock Key", Platform: "mock", Key: "mock-api-key-for-testing", IsDefault: true,
		}); err != nil {
			return err
		}
		_, err = q.V2CreateKeyShare(ctx, db.V2CreateKeyShareParams{CreatedBy: user, ApiKeyID: id, UserID: uuid.NullUUID{UUID: user, Valid: true}})
		return err
	})
	if err != nil {
		log.Warn("failed to create the dev mock key", "user_id", user, "error", err)
	}
}

// preseedDevGames creates a game from every demo game file, owned by the dev
// accounts in turn, every second one public. Only on a fresh instance, while
// no dev account owns a game: games deleted or changed in development stay so.
func preseedDevGames(ctx context.Context) {
	q := store.Queries()
	slots := devUserSlots()
	for _, slot := range slots {
		owned, err := q.V2GamesCreatedBy(ctx, uuid.NullUUID{UUID: slot.id, Valid: true})
		if err != nil || len(owned) > 0 {
			return
		}
	}
	paths, err := DevGameFiles()
	if err != nil || len(paths) == 0 {
		log.Warn("no demo games to seed", "error", err)
		return
	}
	now := time.Now()
	for i, path := range paths {
		owner := slots[i%len(slots)].id
		public := devGamePublic(i)
		if err := preseedDevGame(ctx, owner, path, public, devGameHistory(filepath.Base(path), public, now)); err != nil {
			log.Warn("failed to seed demo game", "file", filepath.Base(path), "user_id", owner, "error", err)
		}
	}
}

// preseedDevGame creates a game of owner's from the game file at path, with
// its visibility and past.
func preseedDevGame(ctx context.Context, owner uuid.UUID, path string, public bool, history gameHistory) error {
	file, err := readGameFile(path)
	if err != nil {
		return err
	}
	g := gameRow{
		name: strings.TrimSpace(file.Name), description: file.Description, genre: file.Genre, theme: file.Theme, public: public,
		scenario: file.SystemMessageScenario, gameStart: file.SystemMessageGameStart, imageStyle: file.ImageStyle,
		voiceStyle:   file.VoiceStyle,
		voices:       file.Voices,
		css:          functional.NormalizeJson(file.CSS, &obj.CSS{}),
		statusFields: functional.NormalizeJson(file.StatusFields, &[]obj.StatusField{}),
	}
	if g.genre == "" {
		g.genre = string(engine.GenreAdventure) // a file without a genre is an adventure
	}
	if err := g.validate(); err != nil {
		return err
	}
	id := uuid.New()
	return store.InTx(ctx, func(q *db.Queries) error {
		if err := q.V2CreateGame(ctx, db.V2CreateGameParams{
			ID: id, CreatedBy: owner, Name: g.name, Description: g.description, Genre: g.genre, Theme: sqlString(g.theme),
			Public: g.public, Scenario: g.scenario, GameStart: g.gameStart, ImageStyle: g.imageStyle, VoiceStyle: g.voiceStyle, Voices: voicesJSON(g.voices), Css: g.css, StatusFields: g.statusFields,
		}); err != nil {
			return err
		}
		return q.V2SeedGameHistory(ctx, db.V2SeedGameHistoryParams{ID: id, CreatedAt: history.createdAt, ModifiedAt: history.modifiedAt, PlayCount: history.playCount})
	})
}

// devGamesDir holds the demo game files, relative to the checkout root.
var devGamesDir = filepath.Join("testing", "testdata", "games")

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

// The platforms a developer may keep a key for, in the order they are
// seeded: the first one seeded becomes the dev admin's default key.
var devKeyPlatforms = []string{"openai", "mistral"}

// devKeyPlaceholder is what the CLI writes into a fresh config file; it is
// no key.
const devKeyPlaceholder = "your-api-key-here"

// DevPlatformKey is the developer's key for a platform from
// ~/.chatgamelab/config.yaml, the file the engine's dev tools read, or ""
// when there is none.
func DevPlatformKey(platform string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".chatgamelab", "config.yaml")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var cfg struct {
		Platforms map[string]struct {
			APIKey string `yaml:"apikey"`
		} `yaml:"platforms"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if key := cfg.Platforms[platform].APIKey; key != devKeyPlaceholder {
		return key, nil
	}
	return "", nil
}

// preseedDevPlatformKeys gives the dev account a key for every platform the
// developer keeps one for and the account has none on yet; the first one
// added becomes its default, the mock key stays for free runs. The first of
// its keys in devKeyPlatforms' order becomes the site's free-use key while
// the site has none, so every dev account plays and draws covers on a real
// platform.
//
// Never under go test: the integration tests run the preseed too, and must
// not spend the developer's money or find keys a test expects absent.
func preseedDevPlatformKeys(ctx context.Context, user uuid.UUID) {
	if testing.Testing() {
		return
	}
	q := store.Queries()
	own, err := q.V2OwnKeyPlatforms(ctx, user)
	if err != nil {
		return
	}
	keys := map[string]uuid.UUID{}
	for _, k := range own {
		keys[k.Platform] = k.ID
	}
	madeDefault := false
	for _, platform := range devKeyPlatforms {
		if _, ok := keys[platform]; ok {
			continue
		}
		secret, err := DevPlatformKey(platform)
		if err != nil {
			log.Warn("failed to read the dev config", "error", err)
			return
		}
		if secret == "" {
			continue
		}
		id := uuid.New()
		err = store.InTx(ctx, func(q *db.Queries) error {
			if err := q.V2CreateKey(ctx, db.V2CreateKeyParams{ID: id, Owner: user, Name: "Dev " + platform + " Key", Platform: platform, Key: secret}); err != nil {
				return err
			}
			if _, err := q.V2CreateKeyShare(ctx, db.V2CreateKeyShareParams{CreatedBy: user, ApiKeyID: id, UserID: uuid.NullUUID{UUID: user, Valid: true}}); err != nil {
				return err
			}
			if madeDefault {
				return nil
			}
			if err := q.V2ClearDefaultKey(ctx, user); err != nil {
				return err
			}
			return q.V2SetDefaultKey(ctx, id)
		})
		if err != nil {
			log.Warn("failed to create dev API key", "platform", platform, "error", err)
			continue
		}
		keys[platform], madeDefault = id, true
	}
	settings, err := q.V2GetSystemSettings(ctx)
	if err != nil || settings.FreeUseApiKeyID.Valid {
		return
	}
	for _, platform := range devKeyPlatforms {
		if id, ok := keys[platform]; ok {
			if err := q.V2UpdateSystemSettingsFreeUseApiKey(ctx, uuid.NullUUID{UUID: id, Valid: true}); err != nil {
				log.Warn("failed to set the site's free-use key", "error", err)
			}
			return
		}
	}
}

// A demo game's cover is kept beside its file, foo.yaml's as foo.cover.jpg:
// the 600-pixel JPEG a cover is stored at, drawn once by cmd/seedcovers.
const devCoverSuffix = ".cover.jpg"

// DevGameFiles lists the demo game files, in file name order.
func DevGameFiles() ([]string, error) {
	dir, err := checkoutPath(devGamesDir)
	if err != nil {
		return nil, err
	}
	return filepath.Glob(filepath.Join(dir, "*.yaml"))
}

// DevCoverPath is where the cover of the demo game file at path is kept.
func DevCoverPath(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + devCoverSuffix
}

// ReadGameFile reads a demo game file.
func ReadGameFile(path string) (obj.GameFile, error) { return readGameFile(path) }

// preseedDevCovers gives every demo game without a cover the one kept beside
// its file, if any, matching the dev accounts' games to the files by name; a
// game with a cover keeps it.
func preseedDevCovers(ctx context.Context) {
	paths, err := DevGameFiles()
	if err != nil || len(paths) == 0 {
		return
	}
	q := store.Queries()
	type owned struct{ id, owner uuid.UUID }
	byName := map[string][]owned{}
	for _, slot := range devUserSlots() {
		games, err := q.V2GameNamesCreatedBy(ctx, uuid.NullUUID{UUID: slot.id, Valid: true})
		if err != nil {
			log.Warn("failed to read dev games", "user_id", slot.id, "error", err)
			return
		}
		for _, g := range games {
			byName[g.Name] = append(byName[g.Name], owned{g.ID, slot.id})
		}
	}
	for _, path := range paths {
		image, err := os.ReadFile(DevCoverPath(path))
		if err != nil {
			continue // no cover drawn for this game yet
		}
		file, err := readGameFile(path)
		if err != nil {
			continue
		}
		for _, game := range byName[file.Name] {
			if _, err := q.V2GameCover(ctx, game.id); !errors.Is(err, sql.ErrNoRows) {
				continue // has a cover, or the lookup failed
			}
			if err := q.V2SetGameCover(ctx, db.V2SetGameCoverParams{GameID: game.id, Image: image, CreatedBy: game.owner}); err != nil {
				log.Warn("failed to seed demo cover", "game_id", game.id, "error", err)
			}
		}
	}
}
