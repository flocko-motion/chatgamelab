// package: obj / core domain data model
// type:    data
// job:     defines the shared domain structs (users, games, API keys, etc.) exchanged across layers
// limits:  data shapes and light helpers only; no persistence or business logic (-> db, game)
package obj

import (
	"time"

	"github.com/google/uuid"
)

// Meta holds creation and modification audit fields common to most entities.
type Meta struct {
	CreatedBy  uuid.NullUUID `json:"createdBy"`
	CreatedAt  *time.Time    `json:"createdAt"`
	ModifiedBy uuid.NullUUID `json:"modifiedBy"`
	ModifiedAt *time.Time    `json:"modifiedAt"`
}

// User represents an application user with role, API keys, and preferences.
type User struct {
	ID            uuid.UUID     `json:"id"`
	Meta          Meta          `json:"meta"`
	Name          string        `json:"name"`
	Email         *string       `json:"email"`
	DeletedAt     *time.Time    `json:"deletedAt"`
	Auth0Id       *string       `json:"auth0Id"`
	Role          *UserRole     `json:"role"`
	ApiKeys       []ApiKeyShare `json:"apiKeys" swaggerignore:"true"`
	AiQualityTier *string       `json:"aiQualityTier,omitempty"` // high/medium/low, nil = server default
	Language      string        `json:"language"`                // ISO 639-1 language code (e.g., "en", "de", "fr")
	// AgeGroup cohort:
	//   "u13"  = 13-17, no parental consent on file (strictest; default when unknown)
	//   "u13p" = 13-17, with parental consent on file
	//   "u18"  = 18+
	//   nil    = guest / unknown
	AgeGroup *string `json:"ageGroup,omitempty"`
}

// UserStats contains aggregated statistics for a user
type UserStats struct {
	GamesCreated      int `json:"gamesCreated"`
	TotalPlaysOnGames int `json:"totalPlaysOnGames"`
}

// Institution represents an organisation with members and shared settings.
type Institution struct {
	ID                   uuid.UUID           `json:"id"`
	Meta                 Meta                `json:"meta"`
	Name                 string              `json:"name"`
	Members              []InstitutionMember `json:"members,omitempty"`
	FreeUseApiKeyShareID *uuid.UUID          `json:"freeUseApiKeyShareId,omitempty"`
	FreeUseAiQualityTier *string             `json:"freeUseAiQualityTier,omitempty"` // high/medium/low, nil = server default
	PromptConstraints    *string             `json:"promptConstraints,omitempty"`
}

// InstitutionMember represents a user's membership and role within an institution.
type InstitutionMember struct {
	UserID uuid.UUID `json:"userId"`
	Name   string    `json:"name"`
	Email  *string   `json:"email,omitempty"`
	Role   Role      `json:"role"`
}

// BackupLog is one entry of the database backup logbook, written by docker/db/backup.sh
// after each run. The server only reads these; it never triggers a backup itself.
type BackupLog struct {
	ID         uuid.UUID `json:"id"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt"`
	Success    bool      `json:"success"`
	Filename   *string   `json:"filename,omitempty"`
	SizeBytes  *int64    `json:"sizeBytes,omitempty"`
	Error      *string   `json:"error,omitempty"` // failure reason, nil on success
}

// SystemSettings holds server-wide configuration such as default AI tiers and prompt constraints.
type SystemSettings struct {
	ID                    uuid.UUID  `json:"id"`
	CreatedAt             *time.Time `json:"createdAt"`
	ModifiedAt            *time.Time `json:"modifiedAt"`
	DefaultAiQualityTier  string     `json:"defaultAiQualityTier"`           // ultimate server-wide fallback (e.g. "medium")
	FreeUseAiQualityTier  *string    `json:"freeUseAiQualityTier,omitempty"` // tier for system free-use key, nil = use default
	PromptConstraintU13   *string    `json:"promptConstraintU13,omitempty"`  // constraint for users aged 13-17 without parental consent (u13)
	PromptConstraintU13p  *string    `json:"promptConstraintU13p,omitempty"` // constraint for users aged 13-17 with parental consent (u13p)
	PromptConstraintU18   *string    `json:"promptConstraintU18,omitempty"`  // constraint for users aged 18+ (u18)
	FreeUseApiKeyID       *uuid.UUID `json:"freeUseApiKeyId,omitempty"`
	FreeUseApiKeyName     string     `json:"freeUseApiKeyName,omitempty"`
	FreeUseApiKeyPlatform string     `json:"freeUseApiKeyPlatform,omitempty"`
	FreeUseApiKeyWorking  *bool      `json:"freeUseApiKeyWorking,omitempty"`
}

// Role identifies a user's permission level within the system.
type Role string

const (
	RoleAdmin       Role = "admin"
	RoleHead        Role = "head"
	RoleStaff       Role = "staff"
	RoleParticipant Role = "participant"
	RoleIndividual  Role = "individual"
)

// InviteStatus represents the lifecycle state of a role invite.
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusDeclined InviteStatus = "declined"
	InviteStatusExpired  InviteStatus = "expired"
	InviteStatusRevoked  InviteStatus = "revoked"
)

// UserRole binds a user to a role within an institution and optional workshop.
type UserRole struct {
	ID          uuid.UUID    `json:"id"`
	Meta        Meta         `json:"meta"`
	UserID      uuid.UUID    `json:"userId"`
	Role        Role         `json:"role"`
	Institution *Institution `json:"institution"`
	Workshop    *Workshop    `json:"workshop,omitempty"`
}

// Workshop represents a workshop within an institution with its settings and participants.
type Workshop struct {
	ID                   uuid.UUID             `json:"id"`
	Meta                 Meta                  `json:"meta"`
	Name                 string                `json:"name"`
	Institution          *Institution          `json:"institution"`
	Active               bool                  `json:"active"`
	Public               bool                  `json:"public"`
	DefaultApiKeyShareID *uuid.UUID            `json:"defaultApiKeyShareId,omitempty"`
	Participants         []WorkshopParticipant `json:"participants,omitempty"`
	Invites              []UserRoleInvite      `json:"invites,omitempty"`
	// Workshop settings (configured by staff/heads)
	AiQualityTier              *string `json:"aiQualityTier,omitempty"` // high/medium/low, nil = server default
	PromptConstraints          *string `json:"promptConstraints,omitempty"`
	ShowPublicGames            bool    `json:"showPublicGames"`
	ShowOtherParticipantsGames bool    `json:"showOtherParticipantsGames"`
	IsPaused                   bool    `json:"isPaused"`
	AllowGameSharing           bool    `json:"allowGameSharing"`
	// Public page /w/<PublicSlug>, visible while Public is on
	PublicSlug        *string `json:"publicSlug,omitempty"`
	PublicDescription *string `json:"publicDescription,omitempty"`
	// PublicLinks is the further reading listed below the games; nil means none.
	PublicLinks []PublicWorkshopLink `json:"publicLinks,omitempty"`
}

// PublicWorkshopLink is one piece of further reading on a workshop's public page.
type PublicWorkshopLink struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// PublicWorkshopPage is what a visitor of /w/<slug> sees. It names no creators and no institution.
type PublicWorkshopPage struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// PlayAvailable is false when the workshop has no key: no game on the page can be played.
	PlayAvailable bool                 `json:"playAvailable"`
	Games         []PublicWorkshopGame `json:"games"`
	Links         []PublicWorkshopLink `json:"links,omitempty"`
}

// PublicWorkshopGame is one public game on a workshop's public page.
type PublicWorkshopGame struct {
	ID          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Play        *PublicWorkshopPlay `json:"play,omitempty"` // nil without a workshop key or while the game is not playable
}

// PublicWorkshopPlay is the share link through which visitors play a game from the public page.
type PublicWorkshopPlay struct {
	Token     string `json:"token"`
	Remaining int    `json:"remaining"`
	Limit     int    `json:"limit"`
}

// WorkshopParticipant represents a participant enrolled in a workshop.
type WorkshopParticipant struct {
	ID          uuid.UUID `json:"id"`
	Meta        Meta      `json:"meta"`
	WorkshopID  uuid.UUID `json:"workshopId"`
	Name        string    `json:"name"`
	Role        Role      `json:"role"`
	AccessToken string    `json:"accessToken"`
	Active      bool      `json:"active"`
	GamesCount  int       `json:"gamesCount"`
	Permanent   bool      `json:"permanent"`
}

// ApiKey represents a platform API key owned by a user.
type ApiKey struct {
	ID               uuid.UUID `json:"id"`
	Meta             Meta      `json:"meta"`
	Name             string    `json:"name"`
	UserID           uuid.UUID `json:"userId"`
	UserName         string    `json:"userName"`
	Platform         string    `json:"platform"`
	Key              string    `json:"-"`
	KeyShortened     string    `json:"keyShortened"`
	IsDefault        bool      `json:"isDefault"`
	LastUsageSuccess *bool     `json:"lastUsageSuccess"`
	LastErrorCode    *string   `json:"lastErrorCode,omitempty"`
}

// ApiKeyShare represents how an API key is shared with a user, workshop, institution, or game.
// The ApiKey contains owner info (UserID, UserName). The share target is one of:
// - User (for direct user-to-user sharing)
// - Workshop (for workshop-scoped sharing)
// - Institution (for institution-wide sharing)
// - Game (for game sponsoring)
type ApiKeyShare struct {
	ID             uuid.UUID    `json:"id"`
	Meta           Meta         `json:"meta"`
	ApiKeyID       uuid.UUID    `json:"apiKeyId"`
	ApiKey         *ApiKey      `json:"apiKey,omitempty"`
	User           *User        `json:"user,omitempty"`
	Workshop       *Workshop    `json:"workshop,omitempty"`
	Institution    *Institution `json:"institution,omitempty"`
	Game           *Game        `json:"game,omitempty"`
	IsUserDefault  bool         `json:"isUserDefault"`
	IsPrivateShare bool         `json:"isPrivateShare,omitempty"`
	Remaining      *int         `json:"remaining,omitempty"`
	GameShareID    *uuid.UUID   `json:"gameShareId,omitempty"`
}

// AvailableKey represents an API key available to a user for playing a specific game
type AvailableKey struct {
	ShareID   uuid.UUID `json:"shareId"`
	Name      string    `json:"name"`
	Platform  string    `json:"platform"`
	Source    string    `json:"source"` // "sponsor", "workshop", "institution", "personal"
	IsDefault bool      `json:"isDefault"`
}

// Game represents a playable game definition including its system prompts and metadata.
type Game struct {
	ID          uuid.UUID `json:"id" yaml:"id"`
	Meta        Meta      `json:"meta" yaml:"-"`
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description" yaml:"description"`
	Icon        []byte    `json:"icon" yaml:"-"`
	// Optional workshop scope (games can be created within a workshop context)
	WorkshopID *uuid.UUID `json:"workshopId,omitempty" yaml:"-"`
	// Access rights and payments. public = true: discoverable on the website and playable by anyone.
	Public bool `json:"public" yaml:"-"`
	// If public, a sponsored API key share can be provided to pay for any public plays.
	PublicSponsoredApiKeyShareID *uuid.UUID `json:"publicSponsoredApiKeyShareId" yaml:"-"`
	// Game details and system messages for the LLM.
	// What is the game about? How does it work? Player role? World description?
	SystemMessageScenario string `json:"systemMessageScenario" yaml:"system_message_scenario"`
	// How should the game start? First scene? How is the player welcomed?
	SystemMessageGameStart string `json:"systemMessageGameStart" yaml:"system_message_game_start"`
	// What style should the images have?
	ImageStyle string `json:"imageStyle" yaml:"image_style"`
	// Additional CSS for the game, probably generated by the LLM.
	// Should be validated/parsed strictly to avoid arbitrary code execution.
	CSS string `json:"css" yaml:"css"`
	// The status fields available to the LLM, shaping the JSON format for status.
	StatusFields string `json:"statusFields" yaml:"status_fields"`
	// Name of the engine theme the game is played in; nil means the engine's default.
	Theme *string `json:"theme,omitempty" yaml:"-"`
	// Quick start: pre-generated first scene of the game.
	// This is generated content (first output after the system message) and may be
	// regenerated from time to time to avoid being too static.
	FirstMessage *string   `json:"firstMessage" yaml:"-"`
	FirstStatus  *string   `json:"firstStatus" yaml:"-"`
	FirstImage   []byte    `json:"firstImage" yaml:"-"`
	Tags         []GameTag `json:"tags" yaml:"-"`
	// Tracking: original creator (for cloned games) and usage statistics
	OriginallyCreatedBy *uuid.UUID `json:"originallyCreatedBy" yaml:"-"`
	PlayCount           int        `json:"playCount" yaml:"-"`
	CloneCount          int        `json:"cloneCount" yaml:"-"`
	// Creator info (populated when fetching games)
	CreatorID   *uuid.UUID `json:"creatorId,omitempty" yaml:"-"`
	CreatorName *string    `json:"creatorName,omitempty" yaml:"-"`
	// Original creator info (populated when fetching games, if originally cloned)
	OriginalCreatorID   *uuid.UUID `json:"originalCreatorId,omitempty" yaml:"-"`
	OriginalCreatorName *string    `json:"originalCreatorName,omitempty" yaml:"-"`
}

// GameShare represents a share link for a game.
// Multiple shares per game are supported (e.g. personal + workshop).
type GameShare struct {
	ID            uuid.UUID  `json:"id"`
	GameID        uuid.UUID  `json:"gameId"`
	Token         string     `json:"token"`
	ApiKeyShareID uuid.UUID  `json:"apiKeyShareId"`
	InstitutionID *uuid.UUID `json:"institutionId"`
	WorkshopID    *uuid.UUID `json:"workshopId"`
	Remaining     *int       `json:"remaining"`
	AiQualityTier *string    `json:"aiQualityTier,omitempty"`
	CreatedBy     *uuid.UUID `json:"createdBy"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// this is just a dummy example for a CSS struct that can be used to have validateable CSS definitions
type CSS struct {
	ColorFg     string `json:"colorFg"`
	ColorBg     string `json:"colorBg"`
	ColorAccent string `json:"colorAccent"`
	ColorButton string `json:"colorButton"`
	Font        string `json:"font"`
}

// GameTag represents a tag label attached to a game.
type GameTag struct {
	ID     uuid.UUID `json:"id"`
	Meta   Meta      `json:"meta"`
	GameID uuid.UUID `json:"gameId"`
	Tag    string    `json:"tag"`
}

// ApiKeyType names the step of the key-resolution chain an API key came from.
const (
	ApiKeyTypeWorkshop           = "workshop"
	ApiKeyTypeSponsor            = "sponsor"
	ApiKeyTypeInstitutionFreeUse = "institution_free_use"
	ApiKeyTypePersonal           = "personal"
	ApiKeyTypeSystemFreeUse      = "system_free_use"
	ApiKeyTypePrivateShare       = "private_share"
)

// StatusField is one named status value of a game, as listed in Game.StatusFields.
type StatusField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
