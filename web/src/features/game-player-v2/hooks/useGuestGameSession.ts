import { useCallback, useMemo } from "react";
import i18next from "i18next";
import { config } from "@/config/env";
import {
  useStreamingSession,
  type SessionAdapter,
  type SessionCreateResult,
  type SessionLoadResult,
  type GameMessageResult,
} from "./useStreamingSession";

const SESSION_STORAGE_KEY_PREFIX = "cgl-guest-session-";

/**
 * Guest game session hook — uses useStreamingSession with plain fetch
 * to the token-gated /api/play/{token}/* endpoints.
 * No authentication required.
 */
export function useGuestGameSession(token: string) {
  const baseUrl = `${config.API_BASE_URL}/play/${token}`;

  // ── Session Storage (recoverability) ─────────────────────────────

  const saveSessionId = useCallback(
    (sessionId: string) => {
      try {
        sessionStorage.setItem(SESSION_STORAGE_KEY_PREFIX + token, sessionId);
      } catch {
        // sessionStorage may be unavailable
      }
    },
    [token],
  );

  const getSavedSessionId = useCallback((): string | null => {
    try {
      return sessionStorage.getItem(SESSION_STORAGE_KEY_PREFIX + token);
    } catch {
      return null;
    }
  }, [token]);

  // ── Adapter ─────────────────────────────────────────────────────

  const adapter: SessionAdapter = useMemo(
    () => ({
      // Guest SSE: no auth headers needed
      getStreamHeaders: async () => ({}),

      createSession: async (): Promise<SessionCreateResult> => {
        const response = await fetch(baseUrl, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            language:
              i18next.resolvedLanguage ??
              i18next.language?.split("-")[0] ??
              "en",
          }),
        });
        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw {
            error: {
              code: errorData.code,
              message:
                errorData.message ||
                `Failed to create session (${response.status})`,
            },
          };
        }
        return response.json();
      },

      sendAction: async (
        sessionId: string,
        message: string,
        statusFields,
      ): Promise<GameMessageResult> => {
        const response = await fetch(`${baseUrl}/sessions/${sessionId}`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ message, statusFields }),
        });
        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw {
            error: {
              code: errorData.code,
              message:
                errorData.message ||
                `Failed to send action (${response.status})`,
            },
          };
        }
        return response.json();
      },

      loadSession: async (sessionId: string): Promise<SessionLoadResult> => {
        const response = await fetch(
          `${baseUrl}/sessions/${sessionId}?messages=all`,
        );
        if (!response.ok) {
          throw new Error("Failed to load session");
        }
        return response.json();
      },

      onSessionCreated: (sessionId: string) => {
        saveSessionId(sessionId);
      },
    }),
    [baseUrl, saveSessionId],
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

  return {
    state,
    startSession,
    sendAction,
    retryLastAction,
    loadExistingSession,
    clearStreamError,
    resetGame,
    getSavedSessionId,
  };
}
