/**
 * Centralized API hooks for TanStack Query.
 *
 * All hooks are organized by domain and re-exported from this file.
 * Import from '@/api/hooks' to use any hook.
 */

// Re-export queryKeys for backwards compatibility
export { queryKeys } from "../queryKeys";

// API Keys
export {
  useApiKeys,
  useCreateApiKey,
  useCheckApiKey,
  useDeleteApiKey,
  useSetDefaultApiKey,
  useInstitutionApiKeys,
  useApiKeyGameShares,
  useShareApiKeyWithInstitution,
  useRemoveInstitutionApiKeyShare,
  useSetInstitutionFreeUseKey,
  useOrganisationApiKeys,
  useOwnApiKeys,
  type OwnApiKey,
  type OwnKeyListSort,
  type OwnKeysAndShares,
  platformFilter,
  type OrganisationKeyListSort,
} from "./useApiKeys";

// Games
export {
  useGames,
  useGamePage,
  useGame,
  useCreateGame,
  useUpdateGame,
  useGameTemplate,
  useDeleteGame,
  useNewVoice,
  useReviewGame,
  useVoiceSample,
  type VoiceSample,
  useExportGameFile,
  useImportGameCover,
  useSponsorGame,
  useRemoveGameSponsor,
  useAddFavorite,
  useRemoveFavorite,
  useApiKeyStatus,
  usePrivateShareStatus,
  useCreateGameShare,
  useRevokePrivateShare,
  useUpdateGameShare,
  useMakeGameCover,
  useCoverDrawing,
  useFetchGame,
  fetchGame,
  gameCoverUrl,
  type GameListFilter,
  genreFilter,
  type GameListSort,
  type GameListParams,
} from "./useGames";

// Sessions
export {
  useUserSessions,
  useGameSessionMap,
  useDeleteSession,
  type UseUserSessionsParams,
} from "./useSessions";

// Users
export {
  useCurrentUser,
  useUserStats,
  useOwnGuestCode,
  useUpdateUser,
  useSetLanguage,
  useUsers,
  useDeleteUser,
  useSetUserRole,
  useRemoveUserRole,
  type UserListFilter,
  type UserListSort,
} from "./useUsers";

// System (platforms, settings, version)
export {
  usePlatforms,
  useGenres,
  useThemes,
  useSystemSettings,
  useUpdateSystemSettings,
  useSetSystemFreeUseKey,
  useVersion,
  useSystemConstraints,
  useUpdates,
  useRequestUpgrade,
  upgradeRunning,
  upgradeHandled,
  checkRunning,
  checkHandled,
  checkMark,
  useCheckForUpdates,
  checkRefused,
} from "./useSystem";
export type { CheckMark, CheckRefused } from "./useSystem";

// Invites
export {
  useRevokeInvite,
  useInvites,
  useCreateOrganisationInvite,
  useMyInvites,
  useAcceptInvite,
  useDeclineInvite,
  fetchInvite,
  acceptInvite,
  type InviteListSort,
} from "./useInvites";

// Workshops
export {
  useWorkshop,
  useCreateWorkshop,
  useUpdateWorkshop,
  useDeleteWorkshop,
  useCreateWorkshopInvite,
  useSetWorkshopApiKey,
  useUpdateParticipant,
  useRemoveParticipant,
  useResetGuestToken,
  useCreateWorkshopEmailInvite,
  useWorkshopList,
  useParticipants,
  useGuestToken,
  usePublicWorkshop,
  usePublicWorkshopGameFile,
  type WorkshopListFilter,
  type WorkshopListSort,
  type ParticipantListSort,
} from "./useWorkshops";

// Signing in and out, and who is calling (plain functions)
export {
  fetchCaller,
  registerAccount,
  checkName,
  signInGuest,
  signInWithAuth0,
  signInDev,
  signOut,
} from "./authentication";

// Workshop Events (SSE)
export { useWorkshopEvents } from "./useWorkshopEvents";

// Games Cache Updater (for SSE events)
export { useGamesCacheUpdater } from "./useGamesCacheUpdater";

// Active Workshop (Workshop Mode)
export { useSetActiveWorkshop } from "./useActiveWorkshop";

// Organisations registering themselves
export {
  useDeclaration,
  useOrganisationConsent,
  useConsentToDeclaration,
  useWithdrawConsent,
  useSetOrganisationSuspension,
  useRecentlyRegisteredInstitutions,
} from "./useOrganisationRegistration";

// Organisations and their members
export {
  useOrganisations,
  useMembers,
  useOrganisation,
  useCreateOrganisation,
  useUpdateOrganisation,
  useDeleteOrganisation,
  useRemoveMember,
  type OrganisationListFilter,
  type OrganisationListSort,
  type MemberListSort,
} from "./useOrganisations";

// An organisation's verified pages, which may frame its embedded games
export {
  useVerifiedPages,
  useCreateVerifiedPage,
  useDeleteVerifiedPage,
  useAllVerifiedPages,
  useApproveVerifiedPage,
  type VerifiedPageListFilter,
  type VerifiedPageListSort,
} from "./useVerifiedPages";

// A game's embed link on another website
export { useGameEmbed, useSetGameEmbed } from "./useGameEmbed";

// Playing: creating and opening engine sessions, what a share link offers
export {
  useCreateSession,
  useOpenSession,
  useFetchShareInfo,
  type NewSession,
} from "./usePlay";
