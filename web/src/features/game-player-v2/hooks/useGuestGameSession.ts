import { useCallback, useMemo } from "react";
import i18next from "i18next";
import { config } from "@/config/env";
import { useAuth } from "@/providers/AuthProvider";
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
  const { getAccessToken } = useAuth();

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

  const adapter: SessionAdapter = useMemo(() => {
    // If the visitor is logged in, send their token so they play the shared game as
    // themselves (own constraint cascade, recently-played). Anonymous visitors send
    // nothing and play as guests. getAccessToken returns null when not logged in.
    const authHeaders = async (): Promise<Record<string, string>> => {
      try {
        const token = await getAccessToken();
        return token ? { Authorization: `Bearer ${token}` } : {};
      } catch {
        return {};
      }
    };
    return {
      getStreamHeaders: authHeaders,

      createSession: async (): Promise<SessionCreateResult> => {
        const response = await fetch(baseUrl, {
          method: "POST",
          headers: { "Content-Type": "application/json", ...(await authHeaders()) },
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
        audio,
      ): Promise<GameMessageResult> => {
        const body: Record<string, unknown> = { message, statusFields };
        if (audio) {
          body.audioBase64 = audio.base64;
          body.audioMimeType = audio.mimeType;
        }
        const response = await fetch(`${baseUrl}/sessions/${sessionId}`, {
          method: "POST",
          headers: { "Content-Type": "application/json", ...(await authHeaders()) },
          body: JSON.stringify(body),
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
          { headers: await authHeaders() },
        );
        if (!response.ok) {
          throw new Error("Failed to load session");
        }
        return response.json();
      },

      onSessionCreated: (sessionId: string) => {
        saveSessionId(sessionId);
      },
    };
  }, [baseUrl, saveSessionId, getAccessToken]);

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
