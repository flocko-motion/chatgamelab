import { useCallback, useEffect, useRef } from "react";
import { showErrorModal } from "@/common/lib/globalErrorModal";
import { useGuestGameSession } from "./useGuestGameSession";
import type { GameInfo, PlayerActionInput } from "../types";
import type { SessionLifecycle } from "./useSessionLifecycle";
import type { GuestStartMode } from "../components/GuestWelcome";

/**
 * Guest session lifecycle — simplified version of useSessionLifecycle for anonymous play.
 *
 * Differences from the authenticated lifecycle:
 * - No useAuth, useWorkshopMode, useApiKeyStatus, useGame
 * - No URL replacement (/sessions/$sessionId)
 * - No workshop event subscriptions
 * - Uses sessionStorage for session recoverability
 * - Always reports apiKeyAvailable = true (sponsor key is server-side)
 */
export function useGuestSessionLifecycle(
  token: string,
  mode: GuestStartMode = "new",
  onBack?: () => void,
): SessionLifecycle {
  const sceneEndRef = useRef<HTMLDivElement>(null);

  const {
    state,
    startSession,
    sendAction,
    retryLastAction,
    loadExistingSession,
    clearStreamError,
    resetGame,
    getSavedSessionId,
  } = useGuestGameSession(token);

  // Start or continue based on mode from welcome screen
  const initAttemptedRef = useRef(false);
  useEffect(() => {
    if (state.phase !== "idle" || initAttemptedRef.current) return;
    initAttemptedRef.current = true;

    if (mode === "continue") {
      const savedSessionId = getSavedSessionId();
      if (savedSessionId) {
        loadExistingSession(savedSessionId);
        return;
      }
    }

    // "new" mode or no saved session — clear storage and start fresh
    try {
      sessionStorage.removeItem(`cgl-guest-session-${token}`);
    } catch {
      /* ignore */
    }
    startSession();
  }, [
    state.phase,
    mode,
    token,
    getSavedSessionId,
    loadExistingSession,
    startSession,
  ]);

  // Auto-scroll on new messages
  const scrollToBottom = useCallback(() => {
    setTimeout(() => {
      sceneEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, 100);
  }, []);

  const prevMessageCountRef = useRef(state.messages.length);
  useEffect(() => {
    if (state.messages.length !== prevMessageCountRef.current) {
      prevMessageCountRef.current = state.messages.length;
      scrollToBottom();
    }
  }, [state.messages, scrollToBottom]);

  // Show global error modal for recoverable mid-game errors
  useEffect(() => {
    if (state.streamError) {
      showErrorModal({
        code: state.streamError.code ?? undefined,
        message: !state.streamError.code
          ? state.streamError.message
          : undefined,
        onDismiss: clearStreamError,
      });
    }
  }, [state.streamError, clearStreamError]);

  const handleSendAction = useCallback(
    async (input: PlayerActionInput) => {
      await sendAction(input);
    },
    [sendAction],
  );

  const handleBack = useCallback(() => {
    if (onBack) {
      onBack();
    } else {
      window.history.back();
    }
  }, [onBack]);

  const displayGame = state.gameInfo as GameInfo | undefined;

  return {
    state,
    startSession,
    sendAction,
    retryLastAction,
    loadExistingSession,
    resetGame,
    gameLoading: false,
    gameError: null,
    gameExists: true,
    displayGame,
    missingFields: [],
    isContinuation: false,
    isInWorkshopContext: false,
    sceneEndRef,
    handleBack,
    handleSendAction,
    isNoApiKeyError: false,
    apiKeyAvailable: true,
    isPausedForUser: false,
  };
}
