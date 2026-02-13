import { useState, useCallback, useRef, useEffect } from "react";
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

// ── Constants ────────────────────────────────────────────────────────────

export const INITIAL_STATE: GamePlayerState = {
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
/** If no SSE data arrives within this window, activate polling as fallback */
const SSE_SILENCE_TIMEOUT = 10_000;
/** Minimum interval between partial image hash updates (throttle re-fetches) */
const PARTIAL_IMAGE_THROTTLE = 4000;

// ── Types ────────────────────────────────────────────────────────────────

/** Headers to attach to the SSE stream request. */
export type StreamHeadersFn = () => Promise<Record<string, string>>;

/**
 * Adapter that the caller provides to bridge auth-specific API calls.
 * Each function receives the current state setter so it can update state.
 */
export interface SessionAdapter {
  /** Build headers for the SSE /messages/{id}/stream request. */
  getStreamHeaders: StreamHeadersFn;

  /** Create a new session. Return the raw session response from the API. */
  createSession: () => Promise<SessionCreateResult>;

  /** Send a player action. Return the raw game message response. */
  sendAction: (
    sessionId: string,
    message: string,
    statusFields: SceneMessage["statusFields"],
  ) => Promise<GameMessageResult>;

  /** Load an existing session by ID. Return the raw session response. */
  loadSession: (sessionId: string) => Promise<SessionLoadResult>;

  /** Called after a session is successfully created (e.g. to invalidate caches). */
  onSessionCreated?: (sessionId: string) => void;
}

export interface SessionCreateResult {
  id?: string;
  gameId?: string;
  gameName?: string;
  gameDescription?: string;
  messages?: RawMessage[];
  theme?: GamePlayerState["theme"];
}

export interface SessionLoadResult {
  id?: string;
  gameId?: string;
  gameName?: string;
  gameDescription?: string;
  apiKeyId?: string;
  messages?: RawMessage[];
  theme?: GamePlayerState["theme"];
}

export interface GameMessageResult {
  id?: string;
  stream?: boolean;
  imagePrompt?: string;
  hasImage?: boolean;
  hasAudio?: boolean;
  statusFields?: SceneMessage["statusFields"];
}

/** Raw message shape from the API (before mapping to SceneMessage). */
export interface RawMessage {
  id?: string;
  stream?: boolean;
  imagePrompt?: string;
  hasImage?: boolean;
  hasAudio?: boolean;
  statusFields?: SceneMessage["statusFields"];
}

// ── Hook ─────────────────────────────────────────────────────────────────

export function useStreamingSession(adapter: SessionAdapter) {
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

  // Keep adapter in a ref so callbacks don't depend on it
  const adapterRef = useRef(adapter);
  adapterRef.current = adapter;

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

        setState((prev) => {
          const msg = prev.messages.find((m) => m.id === messageId);
          if (!msg) return prev;

          const updates: Partial<SceneMessage> = {};
          const stateUpdates: Partial<GamePlayerState> = {};

          // Text: only overwrite if SSE is NOT actively streaming.
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
          ) {
            return prev;
          }

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
      if (activePollingIdRef.current === messageId && pollIntervalRef.current) {
        return;
      }
      stopPolling();
      activePollingIdRef.current = messageId;
      pollErrorCountRef.current = 0;

      pollDelayRef.current = setTimeout(() => {
        pollDelayRef.current = null;
        if (activePollingIdRef.current === messageId) {
          pollMessageStatus(messageId);
        }
      }, 2000);

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

  startPollingRef.current = startPolling;

  const resetSilenceTimer = useCallback(
    (messageId: string) => {
      clearSilenceTimer();
      silenceTimerRef.current = setTimeout(() => {
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

  // ── SSE Streaming ───────────────────────────────────────────────────

  const connectToStream = useCallback(
    async (messageId: string, playerMessageId?: string) => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      const controller = new AbortController();
      abortControllerRef.current = controller;
      lastImageUpdateRef.current = 0;
      const audioChunks: string[] = [];

      try {
        const headers = await adapterRef.current.getStreamHeaders();
        const streamUrl = `${config.API_BASE_URL}/messages/${messageId}/stream`;

        const response = await fetch(streamUrl, {
          headers: { Accept: "text/event-stream", ...headers },
          credentials: "include",
          signal: controller.signal,
        });

        if (!response.ok) {
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
        resetSilenceTimer(messageId);
        const decoder = new TextDecoder();
        let buffer = "";
        let textDone = false;
        let imageDone = false;
        let audioDone = false;

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
                  apiLogger.error("Stream error from backend", {
                    errorCode: chunk.errorCode,
                    error: chunk.error,
                    messageId,
                  });
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

                if (chunk.text) {
                  appendTextToMessage(messageId, chunk.text);
                }

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

                if (chunk.audioData) {
                  audioChunks.push(chunk.audioData);
                }

                if (chunk.textDone) {
                  textDone = true;
                  sseActiveRef.current = false;
                  updateMessage(messageId, { isStreaming: false });
                  setState((prev) => ({
                    ...prev,
                    isWaitingForResponse: false,
                  }));
                }

                if (chunk.audioDone) {
                  audioDone = true;
                  // Decode accumulated base64 chunks into a blob URL
                  try {
                    const binaryStr = audioChunks
                      .map((b64) => atob(b64))
                      .join("");
                    const bytes = new Uint8Array(binaryStr.length);
                    for (let i = 0; i < binaryStr.length; i++) {
                      bytes[i] = binaryStr.charCodeAt(i);
                    }
                    const blob = new Blob([bytes], { type: "audio/mpeg" });
                    const blobUrl = URL.createObjectURL(blob);
                    updateMessage(messageId, {
                      audioStatus: "ready",
                      audioBlobUrl: blobUrl,
                    });
                  } catch (e) {
                    apiLogger.error("Failed to create audio blob", {
                      error: e,
                    });
                    updateMessage(messageId, { audioStatus: "ready" });
                  }
                }

                if (chunk.imageDone) {
                  imageDone = true;
                  updateMessage(messageId, {
                    isImageLoading: false,
                    imageStatus: "complete",
                    imageHash: `sse-${Date.now()}`,
                  });
                }

                // Stream is complete when all channels are done
                if (textDone && imageDone && audioDone) {
                  clearSilenceTimer();
                  stopPolling();
                  return;
                }
              } catch (e) {
                apiLogger.error("Failed to parse stream chunk", {
                  error: e,
                });
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
          apiLogger.error("SSE connection lost, activating polling", {
            error,
            messageId,
          });
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

  // ── Session Actions ─────────────────────────────────────────────────

  const startSession = useCallback(async () => {
    setState((prev) => ({ ...prev, phase: "starting", error: null }));

    try {
      const sessionResponse = await adapterRef.current.createSession();
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
            isImageLoading: !!firstMessage.hasImage,
            audioStatus: firstMessage.hasAudio ? "loading" : undefined,
          },
        ],
        statusFields: firstMessage.statusFields || [],
        isWaitingForResponse: true,
        theme: sessionResponse.theme || null,
      }));

      if (sessionResponse.id) {
        adapterRef.current.onSessionCreated?.(sessionResponse.id);
      }

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
      // Extract message from Error instances or { error: { message } } API shapes
      let message = "Failed to start session";
      if (error instanceof Error) {
        message = error.message;
      } else if (
        error &&
        typeof error === "object" &&
        "error" in error &&
        (error as Record<string, unknown>).error &&
        typeof (error as Record<string, unknown>).error === "object"
      ) {
        const nested = (error as { error: { message?: string } }).error;
        if (nested.message) message = nested.message;
      }
      setState((prev) => ({
        ...prev,
        phase: "error",
        error: message,
        errorObject: error,
      }));
    }
  }, [connectToStream]);

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
        const gameResponse = await adapterRef.current.sendAction(
          state.sessionId,
          message,
          state.statusFields,
        );

        const sceneMessage = mapApiMessageToScene(gameResponse);

        setState((prev) => ({
          ...prev,
          messages: [
            ...prev.messages,
            {
              ...sceneMessage,
              text: "",
              isStreaming: true,
              isImageLoading: !!gameResponse.hasImage,
              audioStatus: gameResponse.hasAudio ? "loading" : undefined,
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
        const isNetworkError =
          error instanceof TypeError && /fetch|network/i.test(error.message);
        const errorCode = isNetworkError
          ? ErrorCodes.NETWORK_ERROR
          : extractRawErrorCode(error) || undefined;
        let errorMessage = "Failed to send action";
        if (error instanceof Error) {
          errorMessage = error.message;
        } else if (
          error &&
          typeof error === "object" &&
          "error" in error &&
          (error as Record<string, unknown>).error &&
          typeof (error as Record<string, unknown>).error === "object"
        ) {
          const nested = (error as { error: { message?: string } }).error;
          if (nested.message) errorMessage = nested.message;
        }
        setState((prev) => ({
          ...prev,
          isWaitingForResponse: false,
          messages: prev.messages.map((msg) =>
            msg.id === playerMessage.id
              ? {
                  ...msg,
                  error: errorMessage,
                  errorCode,
                }
              : msg,
          ),
        }));
      }
    },
    [
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
        const session = await adapterRef.current.loadSession(sessionId);

        const messages = (session.messages || []).map(mapApiMessageToScene);

        const needsNewApiKey = "apiKeyId" in session && !session.apiKeyId;

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
                  ? {
                      ...msg,
                      isImageLoading: !!msg.hasImage,
                      audioStatus: msg.hasAudio ? "loading" : undefined,
                    }
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
          apiLogger.debug(
            "Session has in-progress message, connecting to stream",
            { messageId: lastMessage.id },
          );
          updateMessage(lastMessage.id, { text: "" });
          connectToStream(lastMessage.id);
        } else if (!isInProgress && lastMessage?.id && lastMessage.hasImage) {
          try {
            const statusResp = await fetch(
              `${config.API_BASE_URL}/messages/${lastMessage.id}/status`,
            );
            if (statusResp.ok) {
              const status: MessageStatus = await statusResp.json();
              if (status.imageStatus === "generating") {
                apiLogger.debug(
                  "Session has in-progress image, starting poll",
                  { messageId: lastMessage.id },
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
    [connectToStream, startPolling, updateMessage],
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
    clearStreamError,
    resetGame,
  };
}
