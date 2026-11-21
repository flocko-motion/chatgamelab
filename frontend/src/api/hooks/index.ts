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
  useDeleteApiKey,
  useSetDefaultApiKey,
  useInstitutionApiKeys,
  useApiKeyGameShares,
  useShareApiKeyWithInstitution,
  useRemoveInstitutionApiKeyShare,
  useSetInstitutionFreeUseKey,
} from "./useApiKeys";

// Games
export {
  useGames,
  useGame,
  useCreateGame,
  useUpdateGame,
  useGameTemplate,
  useDeleteGame,
  useExportGameYaml,
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
  gameCoverUrl,
  GAME_LIST_FILTERS,
  GAME_LIST_SORTS,
  type GameListFilter,
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
} from "./useSystem";

// Invites
export {
  useInstitutionInvites,
  useAllInvites,
  useRevokeInvite,
} from "./useInvites";

// Workshops
export {
  useWorkshops,
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
  useAddMemberToWorkshop,
} from "./useWorkshops";

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
