import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from './client';
import { handleApiError } from '../config/queryClient';
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
  return useQuery<ObjApiKeyShare[], HttpxErrorResponse>({
    queryKey: queryKeys.apiKeys,
    queryFn: () => apiClient.apikeys.apikeysList().then(response => response.data),
  });
}

export function useCreateApiKey() {
  const queryClient = useQueryClient();
  
  return useMutation<ObjApiKeyShare, HttpxErrorResponse, RoutesCreateApiKeyRequest>({
    mutationFn: (request) => apiClient.apikeys.postApikeys(request).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

export function useUpdateApiKey() {
  const queryClient = useQueryClient();
  
  return useMutation<ObjApiKeyShare, HttpxErrorResponse, { id: string; request: RoutesShareRequest }>({
    mutationFn: ({ id, request }) => apiClient.apikeys.sharesCreate(id, request).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

// Games hooks
export function useGames() {
  return useQuery<ObjGame[], HttpxErrorResponse>({
    queryKey: queryKeys.games,
    queryFn: () => apiClient.games.gamesList().then(response => response.data),
  });
}

export function useGame(id: string) {
  return useQuery<ObjGame, HttpxErrorResponse>({
    queryKey: [...queryKeys.games, id],
    queryFn: () => apiClient.games.gamesDetail(id).then(response => response.data),
    enabled: !!id,
  });
}

export function useCreateGame() {
  const queryClient = useQueryClient();
  
  return useMutation<ObjGame, HttpxErrorResponse, RoutesCreateGameRequest>({
    mutationFn: (request) => apiClient.games.postGames(request).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
    onError: handleApiError,
  });
}

export function useUpdateGame() {
  const queryClient = useQueryClient();
  
  return useMutation<ObjGame, HttpxErrorResponse, { id: string; game: ObjGame }>({
    mutationFn: ({ id, game }) => apiClient.games.gamesCreate(id, game).then(response => response.data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, id] });
    },
    onError: handleApiError,
  });
}

// Game Sessions hooks
export function useGameSessions(gameId: string) {
  return useQuery<ObjGameSession[], HttpxErrorResponse>({
    queryKey: [...queryKeys.gameSessions, gameId],
    queryFn: () => apiClient.games.sessionsList(gameId).then(response => response.data),
    enabled: !!gameId,
  });
}

export function useCreateGameSession() {
  const queryClient = useQueryClient();
  
  return useMutation<ObjGameSessionMessage, HttpxErrorResponse, { gameId: string; request: RoutesCreateSessionRequest }>({
    mutationFn: ({ gameId, request }) => 
      apiClient.games.sessionsCreate(gameId, request).then(response => response.data),
    onSuccess: (_, { gameId }) => {
      queryClient.invalidateQueries({ queryKey: [...queryKeys.gameSessions, gameId] });
    },
    onError: handleApiError,
  });
}

// Users hooks
export function useUsers() {
  return useQuery<ObjUser[], HttpxErrorResponse>({
    queryKey: queryKeys.users,
    queryFn: () => apiClient.users.usersList().then(response => response.data),
  });
}

export function useCurrentUser() {
  return useQuery<ObjUser, HttpxErrorResponse>({
    queryKey: queryKeys.currentUser,
    queryFn: () => apiClient.users.getUsers().then(response => response.data),
  });
}

export function useUser(id: string) {
  return useQuery<ObjUser, HttpxErrorResponse>({
    queryKey: [...queryKeys.users, id],
    queryFn: () => apiClient.users.usersDetail(id).then(response => response.data),
    enabled: !!id,
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  
  return useMutation<ObjUser, HttpxErrorResponse, { id: string; request: RoutesUserUpdateRequest }>({
    mutationFn: ({ id, request }) => apiClient.users.usersCreate(id, request).then(response => response.data),
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
  
  return useMutation<ObjUser, HttpxErrorResponse, RoutesUsersNewRequest>({
    mutationFn: (request) => apiClient.users.postUsers(request).then(response => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users });
    },
    onError: handleApiError,
  });
}

// Version hook
export function useVersion() {
  return useQuery<RoutesVersionResponse, HttpxErrorResponse>({
    queryKey: queryKeys.version,
    queryFn: () => apiClient.version.versionList().then(response => response.data),
  });
}
