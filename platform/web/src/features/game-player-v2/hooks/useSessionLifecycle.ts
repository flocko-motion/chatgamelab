import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useRouter } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import {
  queryKeys,
  useGame,
  useApiKeyStatus,
  useWorkshopEvents,
  useDeleteSession,
} from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import { useWorkshopMode } from "@/providers/WorkshopModeProvider";
import { extractRawErrorCode } from "@/common/types/errorCodes";
import { showErrorModal } from "@/common/lib/globalErrorModal";
import { useGameSession } from "./useGameSession";
import type { GameInfo, PlayerActionInput } from "../types";

// ── Starting progress tracking ──────────────────────────────────────────
// Persists across route-change re-mounts (games/{id}/play → sessions/{id})

const TOTAL_STARTING_STEPS = 6;

interface CachedProgress {
  steps: boolean[];
  timestamp: string;
}
const progressCache = new Map<string, CachedProgress>();

function makeTraceTimestamp(): string {
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${pad(now.getFullYear() % 100)}${pad(now.getMonth() + 1)}${pad(now.getDate())}${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
}

export interface StartingProgress {
  completedSteps: boolean[];
  sessionId: string | null;
  gameId: string | null;
  traceTimestamp: string;
}

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

  // Starting progress (dots + trace ID)
  startingProgress: StartingProgress;
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

  const deleteSession = useDeleteSession();

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

  // Load existing session (continuation or after creation)
  // Only load when we're actually on the /sessions/{id} route
  const currentPath = router.state.location.pathname;
  const isOnSessionRoute = currentPath.startsWith('/sessions/');
  const loadAttemptedRef = useRef<string | null>(null);

  useEffect(() => {
    // Only load if:
    // 1. We have a sessionId in params
    // 2. State is idle (not already loaded/loading)
    // 3. We're actually on the /sessions/{id} route (not /games/{id}/play)
    // 4. We haven't already attempted to load this session (prevents React StrictMode double-invoke)
    if (sessionId && state.phase === "idle" && isOnSessionRoute && loadAttemptedRef.current !== sessionId) {
      loadAttemptedRef.current = sessionId;
      loadExistingSession(sessionId);
    }
  }, [sessionId, state.phase, loadExistingSession, currentPath, isOnSessionRoute]);

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
    // Navigate when sessionId is set and we're not on the session route yet
    if (state.sessionId && !isOnSessionRoute) {
      urlReplacedRef.current = true;
      navigate({
        to: "/sessions/$sessionId",
        params: { sessionId: state.sessionId },
        replace: true,
      });
    }
  }, [isContinuation, state.sessionId, navigate, isOnSessionRoute]);

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

  // Show global error modal for recoverable mid-game errors (AI errors, send failures)
  useEffect(() => {
    if (state.streamError) {
      const isInitFailure = state.streamError.isInitFailure;
      const sessionIdToDelete = isInitFailure ? state.sessionId : null;

      showErrorModal({
        code: state.streamError.code ?? undefined,
        message: !state.streamError.code
          ? state.streamError.message
          : undefined,
        onDismiss: () => {
          clearStreamError();
          if (isInitFailure && sessionIdToDelete) {
            // Delete the broken empty session and go back
            deleteSession.mutate(sessionIdToDelete);
            resetGame();
            handleBack();
          }
        },
      });
    }
  }, [state.streamError, clearStreamError, state.sessionId, deleteSession, resetGame, handleBack]);

  const handleSendAction = useCallback(
    async (input: PlayerActionInput) => {
      await sendAction(input);
    },
    [sendAction],
  );

  // Check if the error is a "no API key" error
  const isNoApiKeyError =
    state.phase === "error" &&
    extractRawErrorCode(state.errorObject) === "no_api_key";

  // Upfront API key availability check
  const effectiveGameId = gameId || state.gameInfo?.id;
  const { data: apiKeyAvailable = true, isLoading: apiKeyLoading } = useApiKeyStatus(effectiveGameId);

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

  // ── Starting progress (6-dot indicator) ─────────────────────────────
  // Persists across the route-change re-mount via module-level cache.

  const [completedSteps, setCompletedSteps] = useState<boolean[]>(() => {
    if (sessionId) {
      const cached = progressCache.get(sessionId);
      if (cached) return cached.steps;
    }
    return Array(TOTAL_STARTING_STEPS).fill(false);
  });

  const [traceTimestamp] = useState<string>(() => {
    if (sessionId) {
      const cached = progressCache.get(sessionId);
      if (cached) return cached.timestamp;
    }
    return makeTraceTimestamp();
  });

  const markStep = useCallback((index: number) => {
    setCompletedSteps(prev => {
      if (prev[index]) return prev;
      const next = [...prev];
      next[index] = true;
      return next;
    });
  }, []);

  // Dot 0: API key status checked
  useEffect(() => {
    if (!isContinuation && !apiKeyLoading) markStep(0);
  }, [isContinuation, apiKeyLoading, markStep]);

  // Dot 1: Game data loaded
  useEffect(() => {
    if (!isContinuation && !gameLoading && game) markStep(1);
  }, [isContinuation, gameLoading, game, markStep]);

  // Dot 2: Session created (session ID available)
  useEffect(() => {
    if (state.sessionId) markStep(2);
  }, [state.sessionId, markStep]);

  // Dot 3: Navigated to session route
  useEffect(() => {
    if (state.sessionId && isOnSessionRoute) markStep(3);
  }, [state.sessionId, isOnSessionRoute, markStep]);

  // Dot 4: Session load in progress (phase left idle on session route)
  useEffect(() => {
    if (isOnSessionRoute && state.phase !== 'idle') markStep(4);
  }, [isOnSessionRoute, state.phase, markStep]);

  // Dot 5: Session fully loaded
  useEffect(() => {
    if (state.phase === 'playing') markStep(5);
  }, [state.phase, markStep]);

  // Persist progress for cross-remount survival
  useEffect(() => {
    const sid = state.sessionId || sessionId;
    if (sid) {
      progressCache.set(sid, { steps: completedSteps, timestamp: traceTimestamp });
    }
  }, [state.sessionId, sessionId, completedSteps, traceTimestamp]);

  // Clean up cache once game is playing
  useEffect(() => {
    if (state.phase === 'playing') {
      const sid = state.sessionId || sessionId;
      if (sid) setTimeout(() => progressCache.delete(sid), 2000);
    }
  }, [state.phase, state.sessionId, sessionId]);

  const startingProgress = useMemo((): StartingProgress => ({
    completedSteps,
    sessionId: state.sessionId,
    gameId: gameId || null,
    traceTimestamp,
  }), [completedSteps, state.sessionId, gameId, traceTimestamp]);

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
    startingProgress,
  };
}
