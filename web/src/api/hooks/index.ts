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
  useDeleteGame,
  useExportGameYaml,
  useSponsorGame,
  useRemoveGameSponsor,
  useFavoriteGames,
  useAddFavorite,
  useRemoveFavorite,
  useApiKeyStatus,
  usePrivateShareStatus,
  useCreateGameShare,
  useRevokePrivateShare,
  useUpdateGameShare,
  type PrivateShareStatus,
  type EnrichedGameShare,
  type UseGamesParams,
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
  useUpdateUser,
} from "./useUsers";

// System (platforms, settings, version)
export {
  usePlatforms,
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
  useGetParticipantToken,
  useCreateWorkshopEmailInvite,
  useAddMemberToWorkshop,
} from "./useWorkshops";

// Workshop Events (SSE)
export { useWorkshopEvents } from "./useWorkshopEvents";

// Games Cache Updater (for SSE events)
export { useGamesCacheUpdater } from "./useGamesCacheUpdater";

// Active Workshop (Workshop Mode)
export { useSetActiveWorkshop } from "./useActiveWorkshop";
