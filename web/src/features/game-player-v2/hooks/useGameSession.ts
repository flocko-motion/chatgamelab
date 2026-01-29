import { useState, useCallback, useRef, useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { queryKeys } from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import { config } from "@/config/env";
import type { RoutesSessionResponse } from "@/api/generated";
import type {
  SceneMessage,
  StreamChunk,
  GameSessionConfig,
  GamePlayerState,
} from "../types";
import { mapApiMessageToScene } from "../types";

const INITIAL_STATE: GamePlayerState = {
  phase: "selecting-key",
  sessionId: null,
  gameInfo: null,
  messages: [],
  statusFields: [],
  isWaitingForResponse: false,
  error: null,
  errorObject: null,
  theme: null,
};

export function useGameSession(gameId: string) {
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();
  const { getAccessToken } = useAuth();
  const [state, setState] = useState<GamePlayerState>(INITIAL_STATE);
  const abortControllerRef = useRef<AbortController | null>(null);

  const updateMessage = useCallback(
    (messageId: string, update: Partial<SceneMessage>) => {
      setState((prev) => ({
        ...prev,
        messages: prev.messages.map((msg) =>
          msg.id === messageId ? { ...msg, ...update } : msg,
        ),
      }));
    },
    [],
  );

  const appendTextToMessage = useCallback((messageId: string, text: string) => {
    setState((prev) => ({
      ...prev,
      messages: prev.messages.map((msg) =>
        msg.id === messageId ? { ...msg, text: msg.text + text } : msg,
      ),
    }));
  }, []);

  const connectToStream = useCallback(
    async (messageId: string) => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      const controller = new AbortController();
      abortControllerRef.current = controller;

      try {
        const token = await getAccessToken();
        const streamUrl = `${config.API_BASE_URL}/messages/${messageId}/stream`;

        const response = await fetch(streamUrl, {
          headers: {
            Authorization: `Bearer ${token}`,
            Accept: "text/event-stream",
          },
          signal: controller.signal,
        });

        if (!response.ok) {
          throw new Error(`Stream request failed: ${response.status}`);
        }

        const reader = response.body?.getReader();
        if (!reader) {
          throw new Error("No response body");
        }

        const decoder = new TextDecoder();
        let buffer = "";

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() || "";

          for (const line of lines) {
            if (line.startsWith("data: ")) {
              try {
                const data = line.slice(6);
                const chunk: StreamChunk = JSON.parse(data);

                if (chunk.text) {
                  appendTextToMessage(messageId, chunk.text);
                }

                if (chunk.textDone) {
                  updateMessage(messageId, { isStreaming: false });
                  setState((prev) => ({
                    ...prev,
                    isWaitingForResponse: false,
                  }));
                }

                if (chunk.imageDone) {
                  // Image generation complete - polling will detect this
                  updateMessage(messageId, { isImageLoading: false });
                  return;
                }
              } catch (e) {
                console.error("Failed to parse stream chunk:", e);
              }
            }
          }
        }

        setState((prev) => ({ ...prev, isWaitingForResponse: false }));
      } catch (error) {
        if ((error as Error).name !== "AbortError") {
          console.error("Stream error:", error);
          setState((prev) => ({ ...prev, isWaitingForResponse: false }));
        }
      }
    },
    [getAccessToken, appendTextToMessage, updateMessage],
  );

  const startSession = useCallback(
    async (sessionConfig: GameSessionConfig) => {
      setState((prev) => ({ ...prev, phase: "starting", error: null }));

      try {
        const response = await api.games.sessionsCreate(gameId, {
          shareId: sessionConfig.shareId,
          model: sessionConfig.model,
        });

        const sessionResponse = response.data;
        const firstMessage = sessionResponse.messages?.[0];

        if (!firstMessage) {
          throw new Error("No message returned from session creation");
        }

        const sceneMessage = mapApiMessageToScene(firstMessage);

        setState((prev) => ({
          ...prev,
          phase: "playing",
          sessionId: sessionResponse.id || null,
          gameInfo: {
            id: sessionResponse.gameId,
            name: sessionResponse.gameName,
            description: sessionResponse.gameDescription,
          },
          messages: [
            {
              ...sceneMessage,
              text: "",
              isStreaming: true,
              isImageLoading: !!firstMessage.imagePrompt,
            },
          ],
          statusFields: firstMessage.statusFields || [],
          isWaitingForResponse: true,
          theme: sessionResponse.theme || null,
        }));

        // Invalidate caches to refetch sessions and game data
        queryClient.invalidateQueries({
          queryKey: [...queryKeys.gameSessions, gameId],
        });
        queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
        queryClient.invalidateQueries({
          queryKey: [...queryKeys.games, gameId],
        });

        if (firstMessage.id && firstMessage.stream) {
          connectToStream(firstMessage.id);
        } else {
          setState((prev) => ({
            ...prev,
            messages: [sceneMessage],
            isWaitingForResponse: false,
          }));
        }
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to start session";
        setState((prev) => ({
          ...prev,
          phase: "error",
          error: message,
          errorObject: error,
        }));
      }
    },
    [api, gameId, connectToStream, queryClient],
  );

  const sendAction = useCallback(
    async (message: string) => {
      if (!state.sessionId || state.isWaitingForResponse) return;

      const playerMessage: SceneMessage = {
        id: crypto.randomUUID(),
        type: "player",
        text: message,
        timestamp: new Date(),
      };

      setState((prev) => ({
        ...prev,
        messages: [...prev.messages, playerMessage],
        isWaitingForResponse: true,
      }));

      try {
        const response = await api.sessions.sessionsCreate(state.sessionId, {
          message,
          statusFields: state.statusFields, // Send current status for AI context
        });

        const gameResponse = response.data;
        const sceneMessage = mapApiMessageToScene(gameResponse);

        setState((prev) => ({
          ...prev,
          messages: [
            ...prev.messages,
            {
              ...sceneMessage,
              text: "",
              isStreaming: true,
              isImageLoading: !!gameResponse.imagePrompt,
            },
          ],
          // Preserve old status if AI returned empty array
          statusFields: gameResponse.statusFields?.length
            ? gameResponse.statusFields
            : prev.statusFields,
        }));

        if (gameResponse.id && gameResponse.stream) {
          connectToStream(gameResponse.id);
        } else {
          setState((prev) => ({
            ...prev,
            messages: prev.messages.map((msg) =>
              msg.id === sceneMessage.id ? sceneMessage : msg,
            ),
            isWaitingForResponse: false,
          }));
        }
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : "Failed to send action";
        setState((prev) => ({
          ...prev,
          phase: "error",
          isWaitingForResponse: false,
          error: errorMessage,
          errorObject: error,
        }));
      }
    },
    [
      api,
      state.sessionId,
      state.isWaitingForResponse,
      state.statusFields,
      connectToStream,
    ],
  );

  const loadExistingSession = useCallback(
    async (sessionId: string) => {
      setState((prev) => ({ ...prev, phase: "starting", error: null }));

      try {
        const response = await api.sessions.sessionsDetail(sessionId, {
          messages: "all",
        });
        const session: RoutesSessionResponse = response.data;

        const messages = (session.messages || []).map(mapApiMessageToScene);

        // Check if session has no API key (key was deleted)
        const needsNewApiKey = !session.apiKeyId;

        setState((prev) => ({
          ...prev,
          phase: needsNewApiKey ? "needs-api-key" : "playing",
          sessionId,
          gameInfo: {
            id: session.gameId,
            name: session.gameName,
            description: session.gameDescription,
          },
          messages,
          statusFields:
            messages.length > 0
              ? messages[messages.length - 1].statusFields || []
              : [],
          isWaitingForResponse: false,
          theme: session.theme || null,
        }));
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to load session";
        setState((prev) => ({
          ...prev,
          phase: "error",
          error: message,
          errorObject: error,
        }));
      }
    },
    [api],
  );

  const updateSessionApiKey = useCallback(
    async (shareId: string, model?: string) => {
      if (!state.sessionId) return;

      setState((prev) => ({ ...prev, phase: "starting" }));

      try {
        // Update the session with the new API key
        await api.sessions.sessionsPartialUpdate(state.sessionId, {
          shareId,
          model,
        });

        // Transition to playing
        setState((prev) => ({
          ...prev,
          phase: "playing",
        }));

        // Invalidate session query to refresh data
        queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to update session";
        setState((prev) => ({
          ...prev,
          phase: "error",
          error: message,
          errorObject: error,
        }));
      }
    },
    [api, state.sessionId, queryClient],
  );

  const resetGame = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setState(INITIAL_STATE);
  }, []);

  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, []);

  return {
    state,
    startSession,
    sendAction,
    loadExistingSession,
    updateSessionApiKey,
    resetGame,
  };
}
