// package: testutil / the types tests name
// type:    data
// job:     re-exports the generated client's types a test needs to name, so tests check what the API actually sends without importing the client
// limits:  aliases only; add one here when a test needs to name a type
package testutil

import "cgl/apiclient"

// What the API answers.
type (
	User                = apiclient.UserView
	UserRole            = apiclient.UserRoleRef
	UserStats           = apiclient.UserStatsView
	CurrentUser         = apiclient.GetCurrentUserResp
	GuestCode           = apiclient.UserSecretsView
	Game                = apiclient.Game
	JournalEntry        = apiclient.JournalEntry
	GameView            = apiclient.GameView
	VoiceSample         = apiclient.VoiceSampleResp
	Review              = apiclient.ReviewGameResp
	ReviewForm          = apiclient.ReviewGameReq
	GameFile            = apiclient.GameFile
	GameShare           = apiclient.ShareLinkView
	PrivateShareStatus  = apiclient.PrivateShareStatusResp
	GuestGameInfo       = apiclient.GuestGameInfoResp
	AvailableKey        = apiclient.AvailableKey
	GameSession         = apiclient.SessionView
	UserSession         = apiclient.SessionView
	Institution         = apiclient.OrganisationView
	Workshop            = apiclient.WorkshopView
	WorkshopParticipant = apiclient.WorkshopParticipantView
	PublicWorkshopPage  = apiclient.PublicWorkshopPage
	PublicWorkshopLink  = apiclient.PublicWorkshopLink
	Invite              = apiclient.InviteCardView
	InviteDetails       = apiclient.InviteCardView
	AcceptedInvite      = apiclient.AcceptInviteResp
	ApiKey              = apiclient.ApiKeyView
	ApiKeyShare         = apiclient.KeyShareView
	ApiKeyInfo          = apiclient.ApiKeyInfoResp
	ApiKeyGameShare     = apiclient.ShareLinkView
	SystemSettings      = apiclient.SystemSettings
	UpdateState         = apiclient.SchemafUpdateState
	Genre               = apiclient.GenreInfo
	Theme               = apiclient.ThemeInfo
	Snapshot            = apiclient.Snapshot
	Topology            = apiclient.Topology
	SessionEvent        = apiclient.SessionEvent
)

// What a test sends.
type (
	GameUpdate        = apiclient.UpdateGameReq
	Registration      = apiclient.CompleteRegistrationReq
	GameCreate        = apiclient.CreateGameReq
	GameTemplate      = apiclient.GameTemplate
	UserUpdate        = apiclient.UpdateUserReq
	SettingsUpdate    = apiclient.UpdateSystemSettingsReq
	WorkshopInvite    = apiclient.CreateWorkshopInviteReq
	InstitutionCreate = apiclient.CreateInstitutionReq
)

// WorkshopUpdate is a workshop's settings as a test sets them, all at once:
// UpdateWorkshop sends every field (an empty name, and nil pointers, are
// left out), as the app's settings form does.
type WorkshopUpdate struct {
	Name                       string
	Active                     bool
	Public                     bool
	AiQualityTier              *string
	PromptConstraints          *string
	ShowPublicGames            bool
	ShowOtherParticipantsGames bool
	IsPaused                   bool
	AllowGameSharing           bool
	PublicSlug                 *string
	PublicDescription          *string
	PublicLinks                *[]apiclient.PublicWorkshopLink
}

// request is u as the API takes it.
func (u WorkshopUpdate) request() apiclient.UpdateWorkshopReq {
	r := apiclient.UpdateWorkshopReq{
		Active:                     &u.Active,
		Public:                     &u.Public,
		AiQualityTier:              u.AiQualityTier,
		PromptConstraints:          u.PromptConstraints,
		ShowPublicGames:            &u.ShowPublicGames,
		ShowOtherParticipantsGames: &u.ShowOtherParticipantsGames,
		IsPaused:                   &u.IsPaused,
		AllowGameSharing:           &u.AllowGameSharing,
		PublicSlug:                 u.PublicSlug,
		PublicDescription:          u.PublicDescription,
		PublicLinks:                u.PublicLinks,
	}
	if u.Name != "" {
		r.Name = &u.Name
	}
	return r
}

// ApiKeys is the caller's keys and every share of them.
type ApiKeys struct {
	ApiKeys []ApiKey
	Shares  []ApiKeyShare
}
