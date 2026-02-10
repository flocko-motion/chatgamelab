import { useState, useCallback, useRef, useEffect } from "react";
import i18next from "i18next";
import { apiLogger } from "@/config/logger";
import { config } from "@/config/env";
import { extractRawErrorCode, ErrorCodes } from "@/common/types/errorCodes";
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

const POLL_INTERVAL = 3000;
const MAX_POLL_ERRORS = 5;
const SSE_SILENCE_TIMEOUT = 10_000;
const PARTIAL_IMAGE_THROTTLE = 4000;

const SESSION_STORAGE_KEY_PREFIX = "cgl-guest-session-";

/**
 * Guest game session hook — mirrors useGameSession but uses plain fetch
 * to the token-gated /api/play/{token}/* endpoints.
 * No authentication required.
 */
export function useGuestGameSession(token: string) {
  const [state, setState] = useState<GamePlayerState>(INITIAL_STATE);
  const abortControllerRef = useRef<AbortController | null>(null);
  const pollIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollDelayRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pollErrorCountRef = useRef(0);
  const activePollingIdRef = useRef<string | null>(null);
  const sseActiveRef = useRef(false);
  const silenceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const startPollingRef = useRef<(messageId: string) => void>(() => {});
  const lastImageUpdateRef = useRef(0);

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

  // ── Helpers ──────────────────────────────────────────────────────

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

  // ── Message Status Polling ────────────────────────────────────────

  const pollMessageStatus = useCallback(
    async (messageId: string) => {
      try {
        const response = await fetch(
          `${config.API_BASE_URL}/messages/${messageId}/status`,
        );
        if (!response.ok) return;

        const status: MessageStatus = await response.json();
        pollErrorCountRef.current = 0;

        setState((prev) => {
          const msg = prev.messages.find((m) => m.id === messageId);
          if (!msg) return prev;

          const updates: Partial<SceneMessage> = {};
          const stateUpdates: Partial<GamePlayerState> = {};

          if (!sseActiveRef.current && status.text.length > msg.text.length) {
            updates.text = status.text;
          }
          if (status.textDone && msg.isStreaming) {
            updates.isStreaming = false;
            stateUpdates.isWaitingForResponse = false;
          }
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
            if (msg.isImageLoading) updates.isImageLoading = false;
          }
          if (
            status.imageStatus === "error" &&
            status.imageError !== msg.imageErrorCode
          ) {
            updates.imageErrorCode = status.imageError;
          }
          if (
            status.statusFields?.length &&
            JSON.stringify(status.statusFields) !==
              JSON.stringify(msg.statusFields)
          ) {
            updates.statusFields = status.statusFields;
            stateUpdates.statusFields = status.statusFields;
          }

          if (
            Object.keys(updates).length === 0 &&
            Object.keys(stateUpdates).length === 0
          )
            return prev;

          const newMessages = prev.messages.map((m) =>
            m.id === messageId ? { ...m, ...updates } : m,
          );
          return { ...prev, ...stateUpdates, messages: newMessages };
        });

        const imageDone =
          status.imageStatus === "complete" ||
          status.imageStatus === "error" ||
          status.imageStatus === "none";
        if (status.textDone && imageDone) {
          stopPolling();
        }
      } catch {
        pollErrorCountRef.current++;
        if (pollErrorCountRef.current >= MAX_POLL_ERRORS) {
          stopPolling();
        }
      }
    },
    [stopPolling],
  );

  const startPolling = useCallback(
    (messageId: string) => {
      if (activePollingIdRef.current === messageId && pollIntervalRef.current)
        return;
      stopPolling();
      activePollingIdRef.current = messageId;
      pollErrorCountRef.current = 0;

      pollDelayRef.current = setTimeout(() => {
        pollDelayRef.current = null;
        if (activePollingIdRef.current === messageId)
          pollMessageStatus(messageId);
      }, 2000);

      pollIntervalRef.current = setInterval(() => {
        if (activePollingIdRef.current === messageId)
          pollMessageStatus(messageId);
        else stopPolling();
      }, POLL_INTERVAL);
    },
    [pollMessageStatus, stopPolling],
  );

  startPollingRef.current = startPolling;

  const resetSilenceTimer = useCallback(
    (messageId: string) => {
      clearSilenceTimer();
      silenceTimerRef.current = setTimeout(() => {
        if (!pollIntervalRef.current) startPollingRef.current(messageId);
      }, SSE_SILENCE_TIMEOUT);
    },
    [clearSilenceTimer],
  );

  // ── SSE Streaming ─────────────────────────────────────────────────

  const connectToStream = useCallback(
    async (messageId: string, playerMessageId?: string) => {
      if (abortControllerRef.current) abortControllerRef.current.abort();

      const controller = new AbortController();
      abortControllerRef.current = controller;
      lastImageUpdateRef.current = 0;

      try {
        // SSE stream endpoint is public — no auth header needed
        const streamUrl = `${config.API_BASE_URL}/messages/${messageId}/stream`;
        const response = await fetch(streamUrl, {
          headers: { Accept: "text/event-stream" },
          signal: controller.signal,
        });

        if (!response.ok) {
          startPolling(messageId);
          return;
        }

        const reader = response.body?.getReader();
        if (!reader) return;

        sseActiveRef.current = true;
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
                const chunk: StreamChunk = JSON.parse(line.slice(6));
                resetSilenceTimer(messageId);

                if (chunk.error) {
                  sseActiveRef.current = false;
                  clearSilenceTimer();
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

                if (chunk.text) appendTextToMessage(messageId, chunk.text);

                if (chunk.imageData) {
                  const now = Date.now();
                  if (
                    now - lastImageUpdateRef.current >=
                    PARTIAL_IMAGE_THROTTLE
                  ) {
                    lastImageUpdateRef.current = now;
                    updateMessage(messageId, {
                      imageStatus: "generating",
                      imageHash: `partial-${now}`,
                    });
                  }
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

        sseActiveRef.current = false;
        clearSilenceTimer();
        setState((prev) => ({ ...prev, isWaitingForResponse: false }));
      } catch (error) {
        sseActiveRef.current = false;
        clearSilenceTimer();
        if ((error as Error).name !== "AbortError") {
          startPolling(messageId);
        }
      }
    },
    [
      appendTextToMessage,
      updateMessage,
      startPolling,
      stopPolling,
      resetSilenceTimer,
      clearSilenceTimer,
    ],
  );

  // ── Session Actions ───────────────────────────────────────────────

  const startSession = useCallback(async () => {
    setState((prev) => ({ ...prev, phase: "starting", error: null }));

    try {
      const response = await fetch(baseUrl, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          language:
            i18next.resolvedLanguage ?? i18next.language?.split("-")[0] ?? "en",
        }),
      });
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(
          errorData.message || `Failed to create session (${response.status})`,
        );
      }

      const sessionResponse = await response.json();
      const firstMessage = sessionResponse.messages?.[0];

      if (!firstMessage) {
        throw new Error("No message returned from session creation");
      }

      const sceneMessage = mapApiMessageToScene(firstMessage);
      const sessionId = sessionResponse.id;

      // Persist session ID for recoverability
      if (sessionId) saveSessionId(sessionId);

      setState((prev) => ({
        ...prev,
        phase: "playing",
        sessionId: sessionId || null,
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
  }, [baseUrl, connectToStream, saveSessionId]);

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
        const response = await fetch(`${baseUrl}/sessions/${state.sessionId}`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            message,
            statusFields: state.statusFields,
          }),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.message || `Failed to send action (${response.status})`,
          );
        }

        const gameResponse = await response.json();
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
        apiLogger.error("guest sendAction failed", { error });
        const isNetworkError =
          error instanceof TypeError && /fetch|network/i.test(error.message);
        const errorCode = isNetworkError
          ? ErrorCodes.NETWORK_ERROR
          : extractRawErrorCode(error) || undefined;
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
                  errorCode,
                }
              : msg,
          ),
        }));
      }
    },
    [
      baseUrl,
      state.sessionId,
      state.isWaitingForResponse,
      state.statusFields,
      connectToStream,
    ],
  );

  const retryLastAction = useCallback(() => {
    const failedMessage = [...state.messages]
      .reverse()
      .find((msg) => msg.type === "player" && msg.error);
    if (!failedMessage) return;

    setState((prev) => ({
      ...prev,
      messages: prev.messages.filter((msg) => msg.id !== failedMessage.id),
    }));
    setTimeout(() => {
      sendAction(failedMessage.text);
    }, 0);
  }, [state.messages, sendAction]);

  const loadExistingSession = useCallback(
    async (sessionId: string) => {
      setState((prev) => ({ ...prev, phase: "starting", error: null }));

      try {
        const response = await fetch(
          `${baseUrl}/sessions/${sessionId}?messages=all`,
        );
        if (!response.ok) {
          throw new Error("Failed to load session");
        }

        const session = await response.json();
        const messages = (session.messages || []).map(mapApiMessageToScene);
        const lastMessage = messages[messages.length - 1];
        const isInProgress = lastMessage?.isStreaming === true;

        setState((prev) => ({
          ...prev,
          phase: "playing",
          sessionId,
          gameInfo: {
            id: session.gameId,
            name: session.gameName,
            description: session.gameDescription,
          },
          messages: isInProgress
            ? messages.map((msg: SceneMessage, i: number) =>
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

        if (isInProgress && lastMessage?.id) {
          updateMessage(lastMessage.id, { text: "" });
          connectToStream(lastMessage.id);
        } else if (
          !isInProgress &&
          lastMessage?.id &&
          lastMessage.imagePrompt
        ) {
          try {
            const statusResp = await fetch(
              `${config.API_BASE_URL}/messages/${lastMessage.id}/status`,
            );
            if (statusResp.ok) {
              const status: MessageStatus = await statusResp.json();
              if (status.imageStatus === "generating") {
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
            apiLogger.debug("Failed to check image status on rejoin");
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
    [baseUrl, connectToStream, startPolling, updateMessage],
  );

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
      if (abortControllerRef.current) abortControllerRef.current.abort();
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
    clearStreamError,
    resetGame,
    getSavedSessionId,
  };
}
