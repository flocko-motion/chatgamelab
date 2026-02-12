package obj

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TokenUsage tracks token consumption from an API call
type TokenUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

// Add returns a new TokenUsage with the sum of both usages
func (u TokenUsage) Add(other TokenUsage) TokenUsage {
	return TokenUsage{
		InputTokens:  u.InputTokens + other.InputTokens,
		OutputTokens: u.OutputTokens + other.OutputTokens,
		TotalTokens:  u.TotalTokens + other.TotalTokens,
	}
}

type Meta struct {
	CreatedBy  uuid.NullUUID `json:"createdBy"`
	CreatedAt  *time.Time    `json:"createdAt"`
	ModifiedBy uuid.NullUUID `json:"modifiedBy"`
	ModifiedAt *time.Time    `json:"modifiedAt"`
}

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
}

// UserStats contains aggregated statistics for a user
type UserStats struct {
	GamesPlayed       int `json:"gamesPlayed"`
	GamesCreated      int `json:"gamesCreated"`
	MessagesSent      int `json:"messagesSent"`
	TotalPlaysOnGames int `json:"totalPlaysOnGames"`
}

type Institution struct {
	ID                   uuid.UUID           `json:"id"`
	Meta                 Meta                `json:"meta"`
	Name                 string              `json:"name"`
	Members              []InstitutionMember `json:"members,omitempty"`
	FreeUseApiKeyShareID *uuid.UUID          `json:"freeUseApiKeyShareId,omitempty"`
	FreeUseAiQualityTier *string             `json:"freeUseAiQualityTier,omitempty"` // high/medium/low, nil = server default
}

type InstitutionMember struct {
	UserID uuid.UUID `json:"userId"`
	Name   string    `json:"name"`
	Email  *string   `json:"email,omitempty"`
	Role   Role      `json:"role"`
}

type SystemSettings struct {
	ID                    uuid.UUID  `json:"id"`
	CreatedAt             *time.Time `json:"createdAt"`
	ModifiedAt            *time.Time `json:"modifiedAt"`
	DefaultAiQualityTier  string     `json:"defaultAiQualityTier"`           // ultimate server-wide fallback (e.g. "medium")
	FreeUseAiQualityTier  *string    `json:"freeUseAiQualityTier,omitempty"` // tier for system free-use key, nil = use default
	FreeUseApiKeyID       *uuid.UUID `json:"freeUseApiKeyId,omitempty"`
	FreeUseApiKeyName     string     `json:"freeUseApiKeyName,omitempty"`
	FreeUseApiKeyPlatform string     `json:"freeUseApiKeyPlatform,omitempty"`
	FreeUseApiKeyWorking  *bool      `json:"freeUseApiKeyWorking,omitempty"`
}

type Role string

const (
	RoleAdmin       Role = "admin"
	RoleHead        Role = "head"
	RoleStaff       Role = "staff"
	RoleParticipant Role = "participant"
	RoleIndividual  Role = "individual"
)

type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusDeclined InviteStatus = "declined"
	InviteStatusExpired  InviteStatus = "expired"
	InviteStatusRevoked  InviteStatus = "revoked"
)

type UserRole struct {
	ID          uuid.UUID    `json:"id"`
	Meta        Meta         `json:"meta"`
	UserID      uuid.UUID    `json:"userId"`
	Role        Role         `json:"role"`
	Institution *Institution `json:"institution"`
	Workshop    *Workshop    `json:"workshop,omitempty"`
}

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
	ShowPublicGames            bool    `json:"showPublicGames"`
	ShowOtherParticipantsGames bool    `json:"showOtherParticipantsGames"`
	DesignEditingEnabled       bool    `json:"designEditingEnabled"`
	IsPaused                   bool    `json:"isPaused"`
}

type WorkshopParticipant struct {
	ID          uuid.UUID `json:"id"`
	Meta        Meta      `json:"meta"`
	WorkshopID  uuid.UUID `json:"workshopId"`
	Name        string    `json:"name"`
	Role        Role      `json:"role"`
	AccessToken string    `json:"accessToken"`
	Active      bool      `json:"active"`
	GamesCount  int       `json:"gamesCount"`
}

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
}

// ApiKeyShare represents how an API key is shared with a user, workshop, institution, or game.
// The ApiKey contains owner info (UserID, UserName). The share target is one of:
// - User (for direct user-to-user sharing)
// - Workshop (for workshop-scoped sharing)
// - Institution (for institution-wide sharing)
// - Game (for game sponsoring)
type ApiKeyShare struct {
	ID                        uuid.UUID    `json:"id"`
	Meta                      Meta         `json:"meta"`
	ApiKeyID                  uuid.UUID    `json:"apiKeyId"`
	ApiKey                    *ApiKey      `json:"apiKey,omitempty"`
	User                      *User        `json:"user,omitempty"`
	Workshop                  *Workshop    `json:"workshop,omitempty"`
	Institution               *Institution `json:"institution,omitempty"`
	Game                      *Game        `json:"game,omitempty"`
	AllowPublicGameSponsoring bool         `json:"allowPublicGameSponsoring"`
	IsUserDefault             bool         `json:"isUserDefault"`
	IsPrivateShare            bool         `json:"isPrivateShare,omitempty"`
}

// AvailableKey represents an API key available to a user for playing a specific game
type AvailableKey struct {
	ShareID   uuid.UUID `json:"shareId"`
	Name      string    `json:"name"`
	Platform  string    `json:"platform"`
	Source    string    `json:"source"` // "sponsor", "workshop", "institution", "personal"
	IsDefault bool      `json:"isDefault"`
}

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
	// Private share links contain secret random tokens to limit access to the game.
	// They are sponsored, so invited players don't require their own API key.
	PrivateShareHash              *string    `json:"privateShareHash" yaml:"-"`
	PrivateSponsoredApiKeyShareID *uuid.UUID `json:"privateSponsoredApiKeyShareId" yaml:"-"`
	PrivateShareRemaining         *int       `json:"privateShareRemaining" yaml:"-"`
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
	// Optional visual theme override. When set, used directly instead of AI-generating per session.
	Theme *GameTheme `json:"theme,omitempty" yaml:"-"`
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

// this is just a dummy example for a CSS struct that can be used to have validateable CSS definitions
type CSS struct {
	ColorFg     string `json:"colorFg"`
	ColorBg     string `json:"colorBg"`
	ColorAccent string `json:"colorAccent"`
	ColorButton string `json:"colorButton"`
	Font        string `json:"font"`
}

type GameTag struct {
	ID     uuid.UUID `json:"id"`
	Meta   Meta      `json:"meta"`
	GameID uuid.UUID `json:"gameId"`
	Tag    string    `json:"tag"`
}

// GameSession
// A session is created when a user plays a game -> it's the instance of a game.
type GameSession struct {
	ID   uuid.UUID `json:"id"`
	Meta Meta      `json:"meta"`

	GameID          uuid.UUID  `json:"gameId"`
	GameName        string     `json:"gameName"`
	GameDescription string     `json:"gameDescription"`
	UserID          uuid.UUID  `json:"userId"`
	WorkshopID      *uuid.UUID `json:"workshopId,omitempty"`
	UserName        string     `json:"userName"`
	// API key used to pay for this session (sponsored or user-owned), implicitly defines platform.
	// Nullable: key may be deleted, session can continue with a new key.
	ApiKeyID *uuid.UUID `json:"apiKeyId,omitempty"`
	ApiKey   *ApiKey    `json:"apiKey,omitempty"`
	// AI model used for playing.
	AiPlatform string `json:"aiPlatform"`
	AiModel    string `json:"aiModel"`
	// JSON with arbitrary details to be used within that model and within that session.
	AiSession string `json:"aiSession"`

	ImageStyle string `json:"imageStyle"`
	// Language used for this session (ISO 639-1 code), locked at creation time from user preference.
	Language string `json:"language"`
	// Defines the status fields available in the game; copied from game.status_fields at launch.
	StatusFields string `json:"statusFields"`
	// AI-generated visual theme for the game player UI (JSON)
	Theme *GameTheme `json:"theme,omitempty"`
	// Set to true when image generation fails due to organization verification required
	IsOrganisationUnverified bool `json:"isOrganisationUnverified,omitempty"`
}

// GameTheme defines the visual theme for the game player UI.
// This is generated by AI based on the game's description and setting.
// The frontend resolves the preset into a full theme configuration.
// Only minimal overrides are allowed: animation, thinking text, and status emojis.
type GameTheme struct {
	Preset       string            `json:"preset"`                 // preset name (e.g., "space", "medieval", "pirate")
	Animation    string            `json:"animation,omitempty"`    // optional animation override: "none", "stars", "bubbles", etc. Empty = use preset default.
	ThinkingText string            `json:"thinkingText,omitempty"` // optional loading phrase override, e.g. "The tale continues..." Empty = use preset default.
	StatusEmojis map[string]string `json:"statusEmojis,omitempty"` // maps status field names to emoji, e.g. {"Health": "❤️", "Gold": "🪙"}
}

type AiPlatform struct {
	ID             string    `json:"id"`   // technical name without spaces, e.g. "openai"
	Name           string    `json:"name"` // display name e.g. "OpenAI"
	Models         []AiModel `json:"models"`
	SupportsApiKey bool      `json:"supportsApiKey"` // whether this platform supports user API keys
}

type AiModel struct {
	ID            string `json:"id"`                      // generic tier: "high", "medium", "low", "max"
	Name          string `json:"name"`                    // display name e.g. "GPT-5.2"
	Model         string `json:"model"`                   // concrete model ID e.g. "gpt-5.2"
	ImageModel    string `json:"imageModel,omitempty"`    // model used for image generation (if different from Model)
	ImageQuality  string `json:"imageQuality,omitempty"`  // image quality: "high", "medium", "low" (default: "low")
	Description   string `json:"description"`             // tier label e.g. "Premium"
	SupportsImage bool   `json:"supportsImage,omitempty"` // whether this tier generates images
	SupportsAudio bool   `json:"supportsAudio,omitempty"` // whether this tier generates audio (TTS)
}

const (
	AiModelMax      = "max" // highest text model + audio output (OpenAI only)
	AiModelPremium  = "high"
	AiModelBalanced = "medium"
	AiModelEconomy  = "low"
)

// aiModelPriority defines the tier ordering from highest to lowest.
// Higher index = higher priority.
var aiModelPriority = map[string]int{
	AiModelEconomy:  0,
	AiModelBalanced: 1,
	AiModelPremium:  2,
	AiModelMax:      3,
}

// AiModelTierPriority returns the priority of a tier ID (higher = better).
// Returns -1 for unknown tiers.
func AiModelTierPriority(tierID string) int {
	if p, ok := aiModelPriority[tierID]; ok {
		return p
	}
	return -1
}

// ResolveModelWithDowngrade returns the model for the requested tier, or the highest
// available tier below it if the exact tier doesn't exist on this platform.
// Returns nil only if the platform has no models at all.
func (p *AiPlatform) ResolveModelWithDowngrade(tierID string) *AiModel {
	requestedPriority := AiModelTierPriority(tierID)

	// Try exact match first
	var bestMatch *AiModel
	bestPriority := -1
	for i := range p.Models {
		m := &p.Models[i]
		if m.ID == tierID {
			return m
		}
		mp := AiModelTierPriority(m.ID)
		if mp <= requestedPriority && mp > bestPriority {
			bestMatch = m
			bestPriority = mp
		}
	}
	return bestMatch
}

type StatusField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

const (
	GameSessionMessageTypeGame   = "game"   // LLM/game response
	GameSessionMessageTypePlayer = "player" // player action
	GameSessionMessageTypeSystem = "system" // system/context messages
)

// GameSessionMessageAi is the AI-facing JSON structure for player/game messages.
// Status uses a flat map keyed by field name (e.g. {"Health": "100", "Gold": "5"})
// so the AI schema can enforce exact field names and prevent hallucination.
type GameSessionMessageAi struct {
	Type        string            `json:"type"`
	Message     string            `json:"message"`
	Status      map[string]string `json:"status"`
	ImagePrompt *string           `json:"imagePrompt,omitempty"`
}

// ToAiJSON converts a GameSessionMessage to its AI-facing JSON representation.
// Converts internal []StatusField to flat map[string]string for the AI.
func (m *GameSessionMessage) ToAiJSON() string {
	statusMap := make(map[string]string, len(m.StatusFields))
	for _, f := range m.StatusFields {
		statusMap[f.Name] = f.Value
	}
	data, err := json.Marshal(GameSessionMessageAi{
		Type:        m.Type,
		Message:     m.Message,
		Status:      statusMap,
		ImagePrompt: m.ImagePrompt,
	})
	if err != nil {
		return "{}"
	}
	return string(data)
}

type GameSessionMessage struct {
	ID     uuid.UUID `json:"id"`
	Meta   Meta      `json:"meta"`
	Stream bool      `json:"stream"`

	GameSessionID uuid.UUID `json:"gameSessionId"`
	// Sequence number within the session, starting at 1
	Seq int `json:"seq"`
	// player: user message; game: LLM/game response; system: initial system/context messages.
	Type string `json:"type"`
	// Plain text of the scene (system message, player action, or game response).
	Message string `json:"message"`

	PromptStatusUpdate    *string `json:"requestStatusUpdate,omitempty"`
	PromptResponseSchema  *string `json:"requestResponseSchema,omitempty"`
	PromptImageGeneration *string `json:"requestImageGeneration,omitempty"`
	PromptExpandStory     *string `json:"requestExpandStory,omitempty"`
	ResponseRaw           *string `json:"responseRaw,omitempty"`
	URLAnalytics          *string `json:"urlAnalytics,omitempty"`

	// JSON encoded status fields.
	StatusFields []StatusField `json:"statusFields"`
	ImagePrompt  *string       `json:"imagePrompt,omitempty"`
	Image        []byte        `json:"image,omitempty"`
	Audio        []byte        `json:"audio,omitempty"`
	HasImage     bool          `json:"hasImage"` // true when image generation is active for this message
	HasAudio     bool          `json:"hasAudio"` // true when audio narration is active for this message
	TokenUsage   *TokenUsage   `json:"tokenUsage,omitempty"`
}

// GameSessionMessageChunk represents a piece of streamed content (text, image, or audio)
type GameSessionMessageChunk struct {
	Text      string `json:"text,omitempty"`      // Partial text content
	TextDone  bool   `json:"textDone,omitempty"`  // True when text streaming is complete
	ImageData []byte `json:"imageData,omitempty"` // Partial/final image data
	ImageDone bool   `json:"imageDone,omitempty"` // True when image streaming is complete
	AudioData []byte `json:"audioData,omitempty"` // Partial/final audio data (opus)
	AudioDone bool   `json:"audioDone,omitempty"` // True when audio streaming is complete
	Error     string `json:"error,omitempty"`     // Error message if failed
	ErrorCode string `json:"errorCode,omitempty"` // Machine-readable error code (maps to frontend i18n)
}
