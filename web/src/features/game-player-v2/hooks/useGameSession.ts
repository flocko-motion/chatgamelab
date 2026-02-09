import { useState, useCallback, useRef, useEffect } from "react";
import { apiLogger } from "@/config/logger";
import { useQueryClient } from "@tanstack/react-query";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { queryKeys } from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import { config } from "@/config/env";
import type { RoutesSessionResponse } from "@/api/generated";
import { extractRawErrorCode } from "@/common/types/errorCodes";
import type {
  SceneMessage,
  StreamChunk,
  MessageStatus,
  GamePlayerState,
} from "../types";
import { mapApiMessageToScene } from "../types";

const INITIAL_STATE: GamePlayerState = {
  phase: "idle",
  sessionId: null,
  gameInfo: null,
  messages: [],
  statusFields: [],
  isWaitingForResponse: false,
  error: null,
  errorObject: null,
  streamError: null,
  theme: null,
};

const POLL_INTERVAL = 1500;
const MAX_POLL_ERRORS = 5;
/** If no SSE data arrives within this window, activate polling as fallback */
const SSE_SILENCE_TIMEOUT = 10_000;

export function useGameSession(gameId: string) {
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();
  const { getAccessToken } = useAuth();
  const [state, setState] = useState<GamePlayerState>(INITIAL_STATE);
  const abortControllerRef = useRef<AbortController | null>(null);
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollDelayRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pollErrorCountRef = useRef(0);
  // Track the message ID currently being polled
  const activePollingIdRef = useRef<string | null>(null);
  // True while SSE is actively connected and streaming text
  const sseActiveRef = useRef(false);
  // Silence timer: activates polling if no SSE chunk arrives within the timeout
  const silenceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Ref to break circular dependency between resetSilenceTimer and startPolling
  const startPollingRef = useRef<(messageId: string) => void>(() => {});

  // ── Helpers ──────────────────────────────────────────────────────────

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

  const stopPolling = useCallback(() => {
    if (pollDelayRef.current) {
      clearTimeout(pollDelayRef.current);
      pollDelayRef.current = null;
    }
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
    activePollingIdRef.current = null;
    pollErrorCountRef.current = 0;
  }, []);

  const clearSilenceTimer = useCallback(() => {
    if (silenceTimerRef.current) {
      clearTimeout(silenceTimerRef.current);
      silenceTimerRef.current = null;
    }
  }, []);

  // ── Message Status Polling (safety net) ─────────────────────────────

  const pollMessageStatus = useCallback(
    async (messageId: string) => {
      try {
        const response = await fetch(
          `${config.API_BASE_URL}/messages/${messageId}/status`,
        );
        if (!response.ok) return;

        const status: MessageStatus = await response.json();
        pollErrorCountRef.current = 0;

        // Sync state from DB - but be careful not to fight with SSE
        setState((prev) => {
          const msg = prev.messages.find((m) => m.id === messageId);
          if (!msg) return prev;

          const updates: Partial<SceneMessage> = {};
          const stateUpdates: Partial<GamePlayerState> = {};

          // Text: only overwrite if SSE is NOT actively streaming.
          // When SSE is active, it streams char-by-char - polling would cause jumps.
          // When SSE is inactive (dropped or never connected), polling is the fallback.
          if (!sseActiveRef.current && status.text.length > msg.text.length) {
            updates.text = status.text;
          }

          // Text done
          if (status.textDone && msg.isStreaming) {
            updates.isStreaming = false;
            stateUpdates.isWaitingForResponse = false;
          }

          // Image status - only update when actually changed to avoid re-renders
          if (status.imageStatus !== msg.imageStatus) {
            updates.imageStatus = status.imageStatus;
          }
          if (status.imageHash && status.imageHash !== msg.imageHash) {
            updates.imageHash = status.imageHash;
          }
          if (
            status.imageStatus === "complete" ||
            status.imageStatus === "error" ||
            status.imageStatus === "none"
          ) {
            if (msg.isImageLoading) {
              updates.isImageLoading = false;
            }
          }
          if (
            status.imageStatus === "error" &&
            status.imageError !== msg.imageErrorCode
          ) {
            updates.imageErrorCode = status.imageError;
          }

          // Status fields - only update if actually changed
          if (
            status.statusFields?.length &&
            JSON.stringify(status.statusFields) !==
              JSON.stringify(msg.statusFields)
          ) {
            updates.statusFields = status.statusFields;
            stateUpdates.statusFields = status.statusFields;
          }

          // Skip update if nothing changed
          if (
            Object.keys(updates).length === 0 &&
            Object.keys(stateUpdates).length === 0
          ) {
            return prev;
          }

          const newMessages = prev.messages.map((m) =>
            m.id === messageId ? { ...m, ...updates } : m,
          );

          return { ...prev, ...stateUpdates, messages: newMessages };
        });

        // Stop polling when everything is done
        const imageDone =
          status.imageStatus === "complete" ||
          status.imageStatus === "error" ||
          status.imageStatus === "none";

        if (status.textDone && imageDone) {
          apiLogger.debug("Polling complete", { messageId });
          stopPolling();
        }
      } catch {
        pollErrorCountRef.current++;
        if (pollErrorCountRef.current >= MAX_POLL_ERRORS) {
          apiLogger.error("Polling failed too many times, stopping", {
            messageId,
          });
          stopPolling();
        }
      }
    },
    [stopPolling],
  );

  const startPolling = useCallback(
    (messageId: string) => {
      // Already polling this message - don't create duplicate intervals
      if (activePollingIdRef.current === messageId && pollIntervalRef.current) {
        return;
      }
      stopPolling();
      activePollingIdRef.current = messageId;
      pollErrorCountRef.current = 0;

      // Initial poll after a short delay (give SSE a head start)
      pollDelayRef.current = setTimeout(() => {
        pollDelayRef.current = null;
        if (activePollingIdRef.current === messageId) {
          pollMessageStatus(messageId);
        }
      }, 2000);

      // Regular polling interval
      pollIntervalRef.current = setInterval(() => {
        if (activePollingIdRef.current === messageId) {
          pollMessageStatus(messageId);
        } else {
          stopPolling();
        }
      }, POLL_INTERVAL);
    },
    [pollMessageStatus, stopPolling],
  );

  // Keep ref in sync
  startPollingRef.current = startPolling;

  /** Start (or restart) the silence timer for the given message. */
  const resetSilenceTimer = useCallback(
    (messageId: string) => {
      clearSilenceTimer();
      silenceTimerRef.current = setTimeout(() => {
        // No SSE data for SSE_SILENCE_TIMEOUT - activate polling fallback
        if (!pollIntervalRef.current) {
          apiLogger.debug("SSE silence timeout, activating polling fallback", {
            messageId,
          });
          startPollingRef.current(messageId);
        }
      }, SSE_SILENCE_TIMEOUT);
    },
    [clearSilenceTimer],
  );

  // ── SSE Streaming (real-time text) ──────────────────────────────────

  const connectToStream = useCallback(
    async (messageId: string, playerMessageId?: string) => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      const controller = new AbortController();
      abortControllerRef.current = controller;

      try {
        const token = await getAccessToken();
        const streamUrl = `${config.API_BASE_URL}/messages/${messageId}/stream`;

        const headers: Record<string, string> = {
          Accept: "text/event-stream",
        };
        if (token) {
          headers.Authorization = `Bearer ${token}`;
        }

        const response = await fetch(streamUrl, {
          headers,
          credentials: "include",
          signal: controller.signal,
        });

        if (!response.ok) {
          // SSE failed to connect - activate polling as fallback
          apiLogger.error("SSE connection failed, activating polling", {
            status: response.status,
            messageId,
          });
          startPolling(messageId);
          return;
        }

        const reader = response.body?.getReader();
        if (!reader) return;

        sseActiveRef.current = true;
        // Start silence timer - if no data arrives within the timeout, polling kicks in
        resetSilenceTimer(messageId);
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

                // SSE is alive - reset silence timer
                resetSilenceTimer(messageId);

                if (chunk.error) {
                  apiLogger.error("Stream error from backend", {
                    errorCode: chunk.errorCode,
                    error: chunk.error,
                    messageId,
                  });
                  sseActiveRef.current = false;
                  clearSilenceTimer();
                  // Remove the game placeholder message
                  // Mark the PLAYER message as failed (red + retry)
                  setState((prev) => ({
                    ...prev,
                    messages: prev.messages
                      .filter((msg) => msg.id !== messageId)
                      .map((msg) =>
                        msg.id === playerMessageId
                          ? {
                              ...msg,
                              error: chunk.error,
                              errorCode: chunk.errorCode,
                            }
                          : msg,
                      ),
                    isWaitingForResponse: false,
                  }));
                  stopPolling();
                  return;
                }

                if (chunk.text) {
                  appendTextToMessage(messageId, chunk.text);
                }

                // Partial image received - bump imageHash to trigger SceneImage re-fetch
                // (the backend caches partial images, served via /messages/{id}/image)
                if (chunk.imageData) {
                  updateMessage(messageId, {
                    imageStatus: "generating",
                    imageHash: `partial-${Date.now()}`,
                  });
                }

                if (chunk.textDone) {
                  sseActiveRef.current = false;
                  updateMessage(messageId, { isStreaming: false });
                  setState((prev) => ({
                    ...prev,
                    isWaitingForResponse: false,
                  }));
                }

                if (chunk.imageDone) {
                  updateMessage(messageId, {
                    isImageLoading: false,
                    imageStatus: "complete",
                    imageHash: `sse-${Date.now()}`,
                  });
                  // SSE delivered everything - stop polling (if active) and silence timer
                  clearSilenceTimer();
                  stopPolling();
                  return;
                }
              } catch (e) {
                apiLogger.error("Failed to parse stream chunk", { error: e });
              }
            }
          }
        }

        // Stream ended normally
        sseActiveRef.current = false;
        clearSilenceTimer();
        setState((prev) => ({ ...prev, isWaitingForResponse: false }));
      } catch (error) {
        sseActiveRef.current = false;
        clearSilenceTimer();
        if ((error as Error).name !== "AbortError") {
          // SSE dropped - activate polling as fallback
          apiLogger.error("SSE connection lost, activating polling", {
            error,
            messageId,
          });
          startPolling(messageId);
        }
      }
    },
    [
      getAccessToken,
      appendTextToMessage,
      updateMessage,
      startPolling,
      stopPolling,
      resetSilenceTimer,
      clearSilenceTimer,
    ],
  );

  // ── Session Actions ─────────────────────────────────────────────────

  const startSession = useCallback(async () => {
    setState((prev) => ({ ...prev, phase: "starting", error: null }));

    try {
      const response = await api.games.sessionsCreate(gameId, {});

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

      queryClient.invalidateQueries({
        queryKey: [...queryKeys.gameSessions, gameId],
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
      queryClient.invalidateQueries({
        queryKey: [...queryKeys.games, gameId],
      });

      if (firstMessage.id && firstMessage.stream) {
        // No playerMessageId for initial session - system message, no retry needed
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
  }, [api, gameId, connectToStream, queryClient]);

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
        messages: [
          // Clear error from any previously failed player message
          ...prev.messages.map((msg) =>
            msg.error
              ? { ...msg, error: undefined, errorCode: undefined }
              : msg,
          ),
          playerMessage,
        ],
        isWaitingForResponse: true,
      }));

      try {
        const response = await api.sessions.sessionsCreate(state.sessionId, {
          message,
          statusFields: state.statusFields,
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
          statusFields: gameResponse.statusFields?.length
            ? gameResponse.statusFields
            : prev.statusFields,
        }));

        if (gameResponse.id && gameResponse.stream) {
          connectToStream(gameResponse.id, playerMessage.id);
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
        apiLogger.error("sendAction failed", { error });
        const errorCode = extractRawErrorCode(error);
        // Mark the player message as failed (red + retry)
        setState((prev) => ({
          ...prev,
          isWaitingForResponse: false,
          messages: prev.messages.map((msg) =>
            msg.id === playerMessage.id
              ? {
                  ...msg,
                  error:
                    error instanceof Error
                      ? error.message
                      : "Failed to send action",
                  errorCode: errorCode || undefined,
                }
              : msg,
          ),
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

  const retryLastAction = useCallback(() => {
    // Find the last failed player message and resend it
    const failedMessage = [...state.messages]
      .reverse()
      .find((msg) => msg.type === "player" && msg.error);
    if (!failedMessage) return;

    // Remove the failed message, then resend
    setState((prev) => ({
      ...prev,
      messages: prev.messages.filter((msg) => msg.id !== failedMessage.id),
    }));

    // Use setTimeout to ensure state update is applied before resending
    setTimeout(() => {
      sendAction(failedMessage.text);
    }, 0);
  }, [state.messages, sendAction]);

  const loadExistingSession = useCallback(
    async (sessionId: string) => {
      setState((prev) => ({ ...prev, phase: "starting", error: null }));

      try {
        const response = await api.sessions.sessionsDetail(sessionId, {
          messages: "all",
        });
        const session: RoutesSessionResponse = response.data;

        const messages = (session.messages || []).map(mapApiMessageToScene);

        const needsNewApiKey = !session.apiKeyId;

        // Check if the last message is still streaming (backend still processing)
        const lastMessage = messages[messages.length - 1];
        const isInProgress = lastMessage?.isStreaming === true;

        setState((prev) => ({
          ...prev,
          phase: needsNewApiKey ? "needs-api-key" : "playing",
          sessionId,
          gameInfo: {
            id: session.gameId,
            name: session.gameName,
            description: session.gameDescription,
          },
          messages: isInProgress
            ? messages.map((msg, i) =>
                i === messages.length - 1
                  ? { ...msg, isImageLoading: !!msg.imagePrompt }
                  : msg,
              )
            : messages,
          statusFields:
            messages.length > 0
              ? messages[messages.length - 1].statusFields || []
              : [],
          isWaitingForResponse: isInProgress,
          theme: session.theme || null,
        }));

        // If last message is still streaming, try to reconnect to SSE.
        // The backend stream may still be in the registry (no client consumed it yet).
        // connectToStream falls back to polling if SSE returns 404 (stream finished).
        // Reset text to "" because SSE sends incremental deltas that get appended.
        if (isInProgress && lastMessage?.id) {
          apiLogger.debug(
            "Session has in-progress message, connecting to stream",
            {
              messageId: lastMessage.id,
            },
          );
          // Reset text so SSE deltas append cleanly (avoids plotOutline + narrative duplication)
          updateMessage(lastMessage.id, { text: "" });
          connectToStream(lastMessage.id);
        } else if (
          !isInProgress &&
          lastMessage?.id &&
          lastMessage.imagePrompt
        ) {
          // Text is done but image might still be generating, failed, or skipped.
          // mapApiMessageToScene optimistically sets imageStatus="complete", but
          // the image may not be persisted yet. Use the status endpoint as source of truth.
          try {
            const statusResp = await fetch(
              `${config.API_BASE_URL}/messages/${lastMessage.id}/status`,
            );
            if (statusResp.ok) {
              const status: MessageStatus = await statusResp.json();
              if (status.imageStatus === "generating") {
                apiLogger.debug(
                  "Session has in-progress image, starting poll",
                  {
                    messageId: lastMessage.id,
                  },
                );
                updateMessage(lastMessage.id, {
                  imageStatus: "generating",
                  imageHash: status.imageHash || undefined,
                  isImageLoading: true,
                });
                startPolling(lastMessage.id);
              } else if (
                status.imageStatus === "complete" &&
                status.imageHash
              ) {
                updateMessage(lastMessage.id, {
                  imageStatus: "complete",
                  imageHash: status.imageHash,
                });
              } else if (status.imageStatus === "error") {
                updateMessage(lastMessage.id, {
                  imageStatus: "error",
                  imageErrorCode: status.imageError,
                  isImageLoading: false,
                });
              } else if (status.imageStatus === "none") {
                updateMessage(lastMessage.id, {
                  imageStatus: "none",
                  isImageLoading: false,
                });
              }
            }
          } catch {
            // Status check failed - leave optimistic "complete" status, image will 404 gracefully
            apiLogger.debug("Failed to check image status on rejoin", {
              messageId: lastMessage.id,
            });
          }
        }
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
    [api, connectToStream, startPolling, updateMessage],
  );

  const updateSessionApiKey = useCallback(async () => {
    if (!state.sessionId) return;

    setState((prev) => ({ ...prev, phase: "starting" }));

    try {
      await api.sessions.sessionsPartialUpdate(state.sessionId);

      setState((prev) => ({
        ...prev,
        phase: "playing",
      }));

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
  }, [api, state.sessionId, queryClient]);

  const clearStreamError = useCallback(() => {
    setState((prev) => ({ ...prev, streamError: null }));
  }, []);

  const resetGame = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    stopPolling();
    clearSilenceTimer();
    setState(INITIAL_STATE);
  }, [stopPolling, clearSilenceTimer]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      stopPolling();
      clearSilenceTimer();
    };
  }, [stopPolling, clearSilenceTimer]);

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
