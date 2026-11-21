import { api } from "../client/api";
import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import type {
  Error as ApiError,
  SessionDeletedResp,
  SessionView,
} from "../generated/api.gen";

// User Sessions hooks (last played)
export interface UseUserSessionsParams {
  search?: string;
  sortBy?: "game" | "model" | "lastPlayed";
}

export function useUserSessions(params?: UseUserSessionsParams) {
  const { search, sortBy } = params || {};

  return useQuery<SessionView[], ApiError>({
    queryKey: [...queryKeys.userSessions, { search, sortBy }],
    queryFn: () =>
      api.getApiSessions({ search, sortBy })
        .then((response) => response.data),
  });
}

// Hook to get a map of gameId -> session for quick lookup
export function useGameSessionMap() {
  const { data: sessions, isLoading, error } = useUserSessions();

  const sessionMap = useMemo(() => {
    if (!sessions) return new Map<string, SessionView>();
    const map = new Map<string, SessionView>();
    for (const session of sessions) {
      if (session.gameId) {
        map.set(session.gameId, session);
      }
    }
    return map;
  }, [sessions]);

  return { sessionMap, isLoading, error };
}

export function useDeleteSession() {
  const queryClient = useQueryClient();

  return useMutation<SessionDeletedResp, ApiError, string>({
    mutationFn: (id) =>
      api.deleteApiSessionsId(id).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
      queryClient.invalidateQueries({ queryKey: queryKeys.gameSessions });
    },
    onError: handleApiError,
  });
}
