import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { handleApiError } from '../config/queryClient';
import { useRequiredAuthenticatedApi } from './useAuthenticatedApi';
import { apiClient } from './client';
import type { 
  ObjApiKeyShare, 
  ObjGame, 
  ObjGameSession,
  ObjGameSessionMessage,
  ObjUser,
  HttpxErrorResponse,
  RoutesCreateApiKeyRequest,
  RoutesCreateGameRequest,
  RoutesCreateSessionRequest,
  RoutesUsersNewRequest,
  RoutesShareRequest,
  RoutesUserUpdateRequest,
  RoutesVersionResponse
} from './generated';

// Query keys
export const queryKeys = {
  apiKeys: ['apiKeys'] as const,
  apiKeyShares: ['apiKeyShares'] as const,
  games: ['games'] as const,
  gameSessions: ['gameSessions'] as const,
  users: ['users'] as const,
  currentUser: ['currentUser'] as const,
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

export function useUpdateApiKey() {
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

// Games hooks
export function useGames() {
  const api = useRequiredAuthenticatedApi();
  
  return useQuery<ObjGame[], HttpxErrorResponse>({
    queryKey: queryKeys.games,
    queryFn: () => api.games.gamesList().then(response => response.data),
  });
}

export function useGame(id: string) {
  const api = useRequiredAuthenticatedApi();
  
  return useQuery<ObjGame, HttpxErrorResponse>({
    queryKey: [...queryKeys.games, id],
    queryFn: () => api.games.gamesDetail(id).then(response => response.data),
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

// Game Sessions hooks
export function useGameSessions(gameId: string) {
  const api = useRequiredAuthenticatedApi();
  
  return useQuery<ObjGameSession[], HttpxErrorResponse>({
    queryKey: [...queryKeys.gameSessions, gameId],
    queryFn: () => api.games.sessionsList(gameId).then(response => response.data),
    enabled: !!gameId,
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
