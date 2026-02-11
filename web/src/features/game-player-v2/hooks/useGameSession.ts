import { useCallback, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { queryKeys } from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import type { RoutesSessionResponse } from "@/api/generated";
import {
  useStreamingSession,
  type SessionAdapter,
  type SessionCreateResult,
  type SessionLoadResult,
  type GameMessageResult,
} from "./useStreamingSession";

export function useGameSession(gameId: string) {
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();
  const { getAccessToken } = useAuth();

  const adapter: SessionAdapter = useMemo(
    () => ({
      getStreamHeaders: async () => {
        const token = await getAccessToken();
        const headers: Record<string, string> = {};
        if (token) {
          headers.Authorization = `Bearer ${token}`;
        }
        return headers;
      },

      createSession: async (): Promise<SessionCreateResult> => {
        const response = await api.games.sessionsCreate(gameId, {});
        return response.data;
      },

      sendAction: async (
        sessionId: string,
        message: string,
        statusFields,
      ): Promise<GameMessageResult> => {
        const response = await api.sessions.sessionsCreate(sessionId, {
          message,
          statusFields,
        });
        return response.data;
      },

      loadSession: async (sessionId: string): Promise<SessionLoadResult> => {
        const response = await api.sessions.sessionsDetail(sessionId, {
          messages: "all",
        });
        const session: RoutesSessionResponse = response.data;
        return session;
      },

      onSessionCreated: () => {
        queryClient.invalidateQueries({
          queryKey: [...queryKeys.gameSessions, gameId],
        });
        queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
        queryClient.invalidateQueries({
          queryKey: [...queryKeys.games, gameId],
        });
      },
    }),
    [api, gameId, getAccessToken, queryClient],
  );

  const {
    state,
    startSession,
    sendAction,
    retryLastAction,
    loadExistingSession,
    clearStreamError,
    resetGame,
  } = useStreamingSession(adapter);

  // ── Authenticated-only: update session API key ──────────────────────

  const updateSessionApiKey = useCallback(async () => {
    if (!state.sessionId) return;

    // We need direct setState access — but useStreamingSession doesn't expose it.
    // Instead, we call the API and then reload the session.
    try {
      await api.sessions.sessionsPartialUpdate(state.sessionId);
      queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
      // Reload to transition from "needs-api-key" to "playing"
      loadExistingSession(state.sessionId);
    } catch {
      // Error will be handled by loadExistingSession
    }
  }, [api, state.sessionId, queryClient, loadExistingSession]);

  return {
    state,
    startSession,
    sendAction,
    retryLastAction,
    loadExistingSession,
    updateSessionApiKey,
    clearStreamError,
    resetGame,
  };
}
