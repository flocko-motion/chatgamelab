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
  useInstitutionApiKeys,
  useShareApiKeyWithInstitution,
  useRemoveInstitutionApiKeyShare,
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
  useFavoriteGames,
  useAddFavorite,
  useRemoveFavorite,
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
  useCreateUser,
} from "./useUsers";

// System (platforms, roles, settings, version)
export {
  usePlatforms,
  useRoles,
  useSystemSettings,
  useVersion,
} from "./useSystem";

// Invites
export {
  useInvites,
  useInstitutionInvites,
  useAllInvites,
  useRevokeInvite,
} from "./useInvites";
