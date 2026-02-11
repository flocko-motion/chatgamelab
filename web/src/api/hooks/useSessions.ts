import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { useRequiredAuthenticatedApi } from "../useAuthenticatedApi";
import { queryKeys } from "../queryKeys";
import type {
  ObjGameSession,
  HttpxErrorResponse,
  RoutesSessionResponse,
  DbUserSessionWithGame,
} from "../generated";

// Game Sessions hooks
export function useGameSessions(gameId: string) {
  const api = useRequiredAuthenticatedApi();

  return useQuery<ObjGameSession[], HttpxErrorResponse>({
    queryKey: [...queryKeys.gameSessions, gameId],
    queryFn: () =>
      api.games.sessionsList(gameId).then((response) => response.data),
    enabled: !!gameId,
  });
}

// User Sessions hooks (last played)
export interface UseUserSessionsParams {
  search?: string;
  sortBy?: "game" | "model" | "lastPlayed";
}

export function useUserSessions(params?: UseUserSessionsParams) {
  const api = useRequiredAuthenticatedApi();
  const { search, sortBy } = params || {};

  return useQuery<DbUserSessionWithGame[], HttpxErrorResponse>({
    queryKey: [...queryKeys.userSessions, { search, sortBy }],
    queryFn: () =>
      api.sessions
        .sessionsList({ search, sortBy })
        .then((response) => response.data),
  });
}

// Hook to get a map of gameId -> session for quick lookup
export function useGameSessionMap() {
  const { data: sessions, isLoading, error } = useUserSessions();

  const sessionMap = useMemo(() => {
    if (!sessions) return new Map<string, DbUserSessionWithGame>();
    const map = new Map<string, DbUserSessionWithGame>();
    for (const session of sessions) {
      if (session.gameId) {
        map.set(session.gameId, session);
      }
    }
    return map;
  }, [sessions]);

  return { sessionMap, isLoading, error };
}

export function useCreateGameSession() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<RoutesSessionResponse, HttpxErrorResponse, string>({
    mutationFn: (gameId) =>
      api.games.sessionsCreate(gameId).then((response) => response.data),
    onSuccess: (_, gameId) => {
      queryClient.invalidateQueries({
        queryKey: [...queryKeys.gameSessions, gameId],
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, gameId] });
    },
    onError: handleApiError,
  });
}

export function useDeleteSession() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<Record<string, string>, HttpxErrorResponse, string>({
    mutationFn: (id) =>
      api.sessions.sessionsDelete(id).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
      queryClient.invalidateQueries({ queryKey: queryKeys.gameSessions });
    },
    onError: handleApiError,
  });
}
