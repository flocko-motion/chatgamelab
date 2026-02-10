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
  useShareApiKey,
  useUpdateApiKeyName,
  useDeleteApiKey,
  useSetDefaultApiKey,
  useInstitutionApiKeys,
  useShareApiKeyWithInstitution,
  useRemoveInstitutionApiKeyShare,
  useSetInstitutionFreeUseKey,
  useAvailableKeysForGame,
} from "./useApiKeys";

// Games
export {
  useGames,
  useGame,
  useCreateGame,
  useUpdateGame,
  useDeleteGame,
  useCloneGame,
  useExportGameYaml,
  useImportGameYaml,
  useSponsorGame,
  useRemoveGameSponsor,
  useFavoriteGames,
  useAddFavorite,
  useRemoveFavorite,
  useApiKeyStatus,
  type UseGamesParams,
} from "./useGames";

// Sessions
export {
  useGameSessions,
  useUserSessions,
  useGameSessionMap,
  useCreateGameSession,
  useDeleteSession,
  type UseUserSessionsParams,
} from "./useSessions";

// Users
export {
  useUsers,
  useCurrentUser,
  useUserStats,
  useUser,
  useUpdateUser,
  useUpdateUserLanguage,
  useCreateUser,
} from "./useUsers";

// System (platforms, roles, settings, version)
export {
  usePlatforms,
  useRoles,
  useSystemSettings,
  useUpdateSystemSettings,
  useSetSystemFreeUseKey,
  useVersion,
} from "./useSystem";

// Invites
export {
  useInvites,
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
} from "./useWorkshops";

// Workshop Events (SSE)
export { useWorkshopEvents } from "./useWorkshopEvents";

// Games Cache Updater (for SSE events)
export { useGamesCacheUpdater } from "./useGamesCacheUpdater";

// Active Workshop (Workshop Mode)
export { useSetActiveWorkshop } from "./useActiveWorkshop";
