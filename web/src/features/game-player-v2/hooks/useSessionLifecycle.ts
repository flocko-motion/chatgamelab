import { useCallback, useEffect, useMemo, useRef } from "react";
import { useNavigate, useRouter } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import {
  queryKeys,
  useGame,
  useApiKeyStatus,
  useWorkshopEvents,
} from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import { useWorkshopMode } from "@/providers/WorkshopModeProvider";
import { extractRawErrorCode } from "@/common/types/errorCodes";
import { showErrorModal } from "@/common/lib/globalErrorModal";
import { useGameSession } from "./useGameSession";
import type { GameInfo, PlayerActionInput } from "../types";

interface UseSessionLifecycleOptions {
  gameId?: string;
  sessionId?: string;
}

export interface SessionLifecycle {
  // Session state (from useGameSession)
  state: ReturnType<typeof useGameSession>["state"];
  startSession: ReturnType<typeof useGameSession>["startSession"];
  sendAction: ReturnType<typeof useGameSession>["sendAction"];
  retryLastAction: ReturnType<typeof useGameSession>["retryLastAction"];
  loadExistingSession: ReturnType<typeof useGameSession>["loadExistingSession"];
  resetGame: ReturnType<typeof useGameSession>["resetGame"];

  // Game data
  gameLoading: boolean;
  gameError: unknown;
  gameExists: boolean;
  displayGame: GameInfo | undefined;
  isContinuation: boolean;
  isInWorkshopContext: boolean;
  missingFields: string[];

  // Scroll
  sceneEndRef: React.RefObject<HTMLDivElement | null>;

  // Navigation
  handleBack: () => void;
  handleSendAction: (input: PlayerActionInput) => Promise<void>;

  // Error detection
  isNoApiKeyError: boolean;

  // API key availability (checked on entry)
  apiKeyAvailable: boolean;

  // Workshop pause state (true when non-staff user and workshop is paused)
  isPausedForUser: boolean;
}

export function useSessionLifecycle({
  gameId,
  sessionId,
}: UseSessionLifecycleOptions): SessionLifecycle {
  const router = useRouter();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const sceneEndRef = useRef<HTMLDivElement>(null);
  const isContinuation = !!sessionId;
  const { isParticipant, backendUser } = useAuth();
  const { isInWorkshopMode, activeWorkshopId } = useWorkshopMode();

  // User is in workshop context if they are a participant OR staff/head/individual in workshop mode
  const isInWorkshopContext = isParticipant || isInWorkshopMode;

  // Determine workshop ID for SSE subscription
  const workshopIdForEvents = isParticipant
    ? backendUser?.role?.workshop?.id
    : isInWorkshopMode
      ? (activeWorkshopId ?? undefined)
      : undefined;

  // Subscribe to workshop SSE events when in workshop context
  useWorkshopEvents({
    workshopId: workshopIdForEvents,
  });

  // Workshop pause: only affects participants and individuals, not head/staff
  const isStaffOrHead =
    backendUser?.role?.role === "head" || backendUser?.role?.role === "staff";
  const isPausedForUser =
    isInWorkshopContext &&
    !isStaffOrHead &&
    (backendUser?.role?.workshop?.isPaused ?? false);

  const {
    data: game,
    isLoading: gameLoading,
    error: gameError,
  } = useGame(isContinuation ? undefined : gameId);

  const {
    state,
    startSession,
    sendAction,
    retryLastAction,
    loadExistingSession,
    updateSessionApiKey,
    clearStreamError,
    resetGame,
  } = useGameSession(gameId || "");

  // Load existing session (continuation)
  useEffect(() => {
    console.log('[SSE-DEBUG] loadExistingSession effect', { sessionId, phase: state.phase, isContinuation });
    if (sessionId && state.phase === "idle") {
      console.log('[SSE-DEBUG] loadExistingSession effect: calling loadExistingSession', { sessionId });
      loadExistingSession(sessionId);
    }
  }, [sessionId, state.phase, loadExistingSession]);

  // Auto-start new sessions: API key is resolved server-side
  const autoStartAttemptedRef = useRef(false);
  useEffect(() => {
    if (
      isContinuation ||
      state.phase !== "idle" ||
      autoStartAttemptedRef.current
    )
      return;
    if (gameLoading || gameError || !game) return;
    autoStartAttemptedRef.current = true;

    startSession();
  }, [isContinuation, state.phase, gameLoading, gameError, game, startSession]);

  // Replace URL with /sessions/$sessionId once a new session is created,
  // so that F5 resumes the same session instead of creating a new one.
  const urlReplacedRef = useRef(false);
  useEffect(() => {
    if (isContinuation || urlReplacedRef.current) return;
    if (state.sessionId && state.phase === "playing") {
      urlReplacedRef.current = true;
      navigate({
        to: "/sessions/$sessionId",
        params: { sessionId: state.sessionId },
        replace: true,
      });
    }
  }, [isContinuation, state.sessionId, state.phase, navigate]);

  // Auto-resolve API key for sessions that lost their key (needs-api-key phase)
  useEffect(() => {
    if (state.phase === "needs-api-key") {
      updateSessionApiKey();
    }
  }, [state.phase, updateSessionApiKey]);

  // Debug: Log received theme
  useEffect(() => {
    if (state.theme) {
      console.log(
        "[GamePlayer] Received theme from session:",
        JSON.stringify(state.theme, null, 2),
      );
    }
  }, [state.theme]);

  // Auto-scroll on new messages
  const scrollToBottom = useCallback(() => {
    setTimeout(() => {
      sceneEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, 100);
  }, []);

  const prevMessageCountRef = useRef(state.messages.length);
  useEffect(() => {
    // Only auto-scroll when a new message is added, not on streaming text updates
    if (state.messages.length !== prevMessageCountRef.current) {
      prevMessageCountRef.current = state.messages.length;
      scrollToBottom();
    }
  }, [state.messages, scrollToBottom]);

  // Show global error modal for recoverable mid-game errors (AI errors, send failures)
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
    // Invalidate queries so the games/sessions lists refresh with any new sessions
    queryClient.invalidateQueries({ queryKey: queryKeys.games });
    queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
    // Navigate back to wherever the user came from (My Games, All Games, Sessions, etc.)
    if (window.history.length > 1) {
      router.history.back();
    } else {
      // Fallback if there's no history (e.g. direct URL access)
      navigate({ to: "/" });
    }
  }, [queryClient, router, navigate]);

  // Check if the error is a "no API key" error
  const isNoApiKeyError =
    state.phase === "error" &&
    extractRawErrorCode(state.errorObject) === "no_api_key";

  // Upfront API key availability check
  const effectiveGameId = gameId || state.gameInfo?.id;
  const { data: apiKeyAvailable = true } = useApiKeyStatus(effectiveGameId);

  const displayGame = (isContinuation ? state.gameInfo : game) as
    | GameInfo
    | undefined;

  // Validate required game fields before allowing play
  const missingFields = useMemo(() => {
    if (!game || isContinuation) return [];
    const missing: string[] = [];
    if (!game.systemMessageScenario?.trim()) missing.push("Scenario");
    if (!game.systemMessageGameStart?.trim()) missing.push("Game Start");
    if (!game.imageStyle?.trim()) missing.push("Image Style");
    return missing;
  }, [game, isContinuation]);

  return {
    state,
    startSession,
    sendAction,
    retryLastAction,
    loadExistingSession,
    resetGame,
    gameLoading,
    gameError,
    gameExists: !!game,
    displayGame,
    missingFields,
    isContinuation,
    isInWorkshopContext,
    sceneEndRef,
    handleBack,
    handleSendAction,
    isNoApiKeyError,
    apiKeyAvailable,
    isPausedForUser,
  };
}
