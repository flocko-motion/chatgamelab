import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useAuth0 } from '@auth0/auth0-react';
import { handleApiError } from '../config/queryClient';
import { useRequiredAuthenticatedApi } from './useAuthenticatedApi';
import { apiClient } from './client';
import type { 
  ObjApiKeyShare,
  ObjAiPlatform,
  ObjGame, 
  ObjGameSession,
  ObjGameSessionMessage,
  ObjUser,
  HttpxErrorResponse,
  RoutesCreateApiKeyRequest,
  RoutesCreateGameRequest,
  RoutesCreateSessionRequest,
  RoutesRolesResponse,
  RoutesUsersNewRequest,
  RoutesShareRequest,
  RoutesUserUpdateRequest,
  RoutesVersionResponse,
  DbUserSessionWithGame
} from './generated';

// Query keys
export const queryKeys = {
  apiKeys: ['apiKeys'] as const,
  apiKeyShares: ['apiKeyShares'] as const,
  platforms: ['platforms'] as const,
  games: ['games'] as const,
  gameSessions: ['gameSessions'] as const,
  userSessions: ['userSessions'] as const,
  users: ['users'] as const,
  currentUser: ['currentUser'] as const,
  roles: ['roles'] as const,
  version: ['version'] as const,
} as const;

// API Keys hooks
export function useApiKeys() {
  const api = useRequiredAuthenticatedApi();
  
  return useQuery<ObjApiKeyShare[], HttpxErrorResponse>({
    queryKey: queryKeys.apiKeys,
    queryFn: () => api.apikeys.apikeysList().then(response => response.data),
  });
}

export function useCreateApiKey() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjApiKeyShare, HttpxErrorResponse, RoutesCreateApiKeyRequest>({
    mutationFn: (request) => api.apikeys.postApikeys(request).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

export function useShareApiKey() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjApiKeyShare, HttpxErrorResponse, { id: string; request: RoutesShareRequest }>({
    mutationFn: ({ id, request }) => api.apikeys.sharesCreate(id, request).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

export function useUpdateApiKeyName() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjApiKeyShare, HttpxErrorResponse, { id: string; name: string }>({
    mutationFn: ({ id, name }) => api.apikeys.apikeysPartialUpdate(id, { name }).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

export function useDeleteApiKey() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjApiKeyShare, HttpxErrorResponse, { id: string; cascade?: boolean }>({
    mutationFn: ({ id, cascade }) => api.apikeys.apikeysDelete(id, { cascade }).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

// Platforms hook (public endpoint, no auth needed)
export function usePlatforms() {
  const api = apiClient;
  
  return useQuery<ObjAiPlatform[], HttpxErrorResponse>({
    queryKey: queryKeys.platforms,
    queryFn: () => api.platforms.platformsList().then((response) => response.data),
  });
}

// Roles hook (public endpoint, no auth needed)
export function useRoles() {
  const api = apiClient;
  
  return useQuery<RoutesRolesResponse, HttpxErrorResponse>({
    queryKey: queryKeys.roles,
    queryFn: () => api.roles.rolesList().then((response) => response.data),
  });
}

// Games hooks
export interface UseGamesParams {
  search?: string;
  sortBy?: 'name' | 'createdAt' | 'modifiedAt';
  sortDir?: 'asc' | 'desc';
  filter?: 'all' | 'own' | 'public' | 'organization' | 'favorites';
}

export function useGames(params?: UseGamesParams) {
  const api = useRequiredAuthenticatedApi();
  const { search, sortBy, sortDir, filter } = params || {};
  
  return useQuery<ObjGame[], HttpxErrorResponse>({
    queryKey: [...queryKeys.games, { search, sortBy, sortDir, filter }],
    queryFn: () => api.games.gamesList({ 
      search: search || undefined,
      sortBy: sortBy || undefined,
      sortDir: sortDir || undefined,
      filter: filter || undefined,
    }).then(response => response.data),
  });
}

export function useGame(id: string | undefined) {
  const api = useRequiredAuthenticatedApi();
  
  return useQuery<ObjGame, HttpxErrorResponse>({
    queryKey: [...queryKeys.games, id],
    queryFn: () => api.games.gamesDetail(id!).then(response => response.data),
    enabled: !!id,
  });
}

export function useCreateGame() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjGame, HttpxErrorResponse, RoutesCreateGameRequest>({
    mutationFn: (request) => api.games.postGames(request).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
    onError: handleApiError,
  });
}

export function useUpdateGame() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjGame, HttpxErrorResponse, { id: string; game: ObjGame }>({
    mutationFn: ({ id, game }) => api.games.gamesCreate(id, game).then(response => response.data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, id] });
    },
    onError: handleApiError,
  });
}

export function useDeleteGame() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjGame, HttpxErrorResponse, string>({
    mutationFn: (id) => api.games.gamesDelete(id).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
    onError: handleApiError,
  });
}

export function useCloneGame() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjGame, HttpxErrorResponse, string>({
    mutationFn: (id) => api.games.cloneCreate(id).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
    onError: handleApiError,
  });
}

export function useExportGameYaml() {
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<string, HttpxErrorResponse, string>({
    mutationFn: async (id) => {
      const response = await api.games.yamlList(id);
      // The response is text/yaml, so we need to get it as text
      return response.data as unknown as string;
    },
    onError: handleApiError,
  });
}

export function useImportGameYaml() {
  const queryClient = useQueryClient();
  const { getAccessTokenSilently } = useAuth0();
  
  return useMutation<ObjGame, HttpxErrorResponse, { id: string; yaml: string }>({
    mutationFn: async ({ id, yaml }) => {
      const token = await getAccessTokenSilently();
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL || ''}/games/${id}/yaml`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/x-yaml',
          'Authorization': `Bearer ${token}`,
        },
        body: yaml, // Send raw YAML, not JSON-stringified
      });
      
      if (!response.ok) {
        const error = await response.json().catch(() => ({ message: 'Import failed' }));
        throw { ...error, status: response.status };
      }
      
      return response.json();
    },
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, id] });
    },
    onError: handleApiError,
  });
}

// Game Sessions hooks
export function useGameSessions(gameId: string) {
  const api = useRequiredAuthenticatedApi();
  
  return useQuery<ObjGameSession[], HttpxErrorResponse>({
    queryKey: [...queryKeys.gameSessions, gameId],
    queryFn: () => api.games.sessionsList(gameId).then(response => response.data),
    enabled: !!gameId,
  });
}

// User Sessions hooks (last played)
export interface UseUserSessionsParams {
  search?: string;
  sortBy?: 'game' | 'model' | 'lastPlayed';
}

export function useUserSessions(params?: UseUserSessionsParams) {
  const api = useRequiredAuthenticatedApi();
  const { search, sortBy } = params || {};
  
  return useQuery<DbUserSessionWithGame[], HttpxErrorResponse>({
    queryKey: [...queryKeys.userSessions, { search, sortBy }],
    queryFn: () => api.sessions.sessionsList({ search, sortBy }).then(response => response.data),
  });
}

export function useCreateGameSession() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjGameSessionMessage, HttpxErrorResponse, { gameId: string; request: RoutesCreateSessionRequest }>({
    mutationFn: ({ gameId, request }) => 
      api.games.sessionsCreate(gameId, request).then(response => response.data),
    onSuccess: (_, { gameId }) => {
      queryClient.invalidateQueries({ queryKey: [...queryKeys.gameSessions, gameId] });
    },
    onError: handleApiError,
  });
}

export function useDeleteSession() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<Record<string, string>, HttpxErrorResponse, string>({
    mutationFn: (id) => api.sessions.sessionsDelete(id).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
      queryClient.invalidateQueries({ queryKey: queryKeys.gameSessions });
    },
    onError: handleApiError,
  });
}

// Users hooks
export function useUsers() {
  const api = useRequiredAuthenticatedApi();
  
  return useQuery<ObjUser[], HttpxErrorResponse>({
    queryKey: queryKeys.users,
    queryFn: () => api.users.usersList().then(response => response.data),
  });
}

export function useCurrentUser() {
  const api = useRequiredAuthenticatedApi();
  
  return useQuery<ObjUser, HttpxErrorResponse>({
    queryKey: queryKeys.currentUser,
    queryFn: () => api.users.getUsers().then(response => response.data),
  });
}

export function useUser(id: string) {
  const api = useRequiredAuthenticatedApi();
  
  return useQuery<ObjUser, HttpxErrorResponse>({
    queryKey: [...queryKeys.users, id],
    queryFn: () => api.users.usersDetail(id).then(response => response.data),
    enabled: !!id,
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjUser, HttpxErrorResponse, { id: string; request: RoutesUserUpdateRequest }>({
    mutationFn: ({ id, request }) => api.users.usersCreate(id, request).then(response => response.data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.users, id] });
      if (id === 'me') {
        queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
      }
    },
    onError: handleApiError,
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();
  
  return useMutation<ObjUser, HttpxErrorResponse, RoutesUsersNewRequest>({
    mutationFn: (request) => api.users.postUsers(request).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users });
    },
    onError: handleApiError,
  });
}

// Version hook (public endpoint, no auth needed)
export function useVersion() {
  const api = apiClient;
  
  return useQuery<RoutesVersionResponse, HttpxErrorResponse>({
    queryKey: queryKeys.version,
    queryFn: () => api.version.versionList().then(response => response.data),
  });
}
