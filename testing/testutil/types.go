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
	GameView            = apiclient.GameView
	GameFile            = apiclient.GameFile
	GameShare           = apiclient.GameShareResp
	PrivateShareStatus  = apiclient.PrivateShareStatusResp
	GuestGameInfo       = apiclient.GuestGameInfoResp
	AvailableKey        = apiclient.AvailableKey
	GameSession         = apiclient.GameSession
	UserSession         = apiclient.UserSessionWithGame
	Institution         = apiclient.Institution
	InstitutionMember   = apiclient.InstitutionMember
	Workshop            = apiclient.Workshop
	WorkshopParticipant = apiclient.WorkshopParticipant
	PublicWorkshopPage  = apiclient.PublicWorkshopPage
	PublicWorkshopLink  = apiclient.PublicWorkshopLink
	Invite              = apiclient.UserRoleInvite
	InviteDetails       = apiclient.InviteResp
	AcceptedInvite      = apiclient.AcceptInviteResp
	ApiKey              = apiclient.ApiKey
	ApiKeyShare         = apiclient.ApiKeyShare
	ApiKeyInfo          = apiclient.ApiKeyInfoResp
	ApiKeys             = apiclient.ApiKeysResp
	ApiKeyGameShare     = apiclient.EnrichedGameShare
	SystemSettings      = apiclient.SystemSettings
	Genre               = apiclient.GenreInfo
	Theme               = apiclient.ThemeInfo
	Snapshot            = apiclient.Snapshot
	Topology            = apiclient.Topology
	SessionEvent        = apiclient.SessionEvent
)

// What a test sends.
type (
	GameUpdate     = apiclient.UpdateGameReq
	Registration   = apiclient.CompleteRegistrationReq
	GameCreate     = apiclient.CreateGameReq
	GameTemplate   = apiclient.GameTemplate
	WorkshopUpdate = apiclient.UpdateWorkshopReq
	UserUpdate     = apiclient.UpdateUserReq
	SettingsUpdate = apiclient.UpdateSystemSettingsReq
	WorkshopInvite = apiclient.CreateWorkshopInviteReq
)
