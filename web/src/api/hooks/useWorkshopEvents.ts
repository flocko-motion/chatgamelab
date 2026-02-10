import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { config } from "@/config/env";
import { queryKeys } from "../queryKeys";
import { uiLogger } from "@/config/logger";
import { useAuth } from "@/providers/AuthProvider";

interface GameEventData {
  gameId: string;
  triggeredBy: string;
}

interface UseWorkshopEventsOptions {
  workshopId: string | undefined;
  enabled?: boolean;
  /** Called when workshop settings are updated - use to refresh backend user */
  onSettingsUpdate?: () => void;
  /** Called when a game is created by another user in the workshop */
  onGameCreated?: (gameId: string) => void;
  /** Called when a game is updated by another user in the workshop */
  onGameUpdated?: (gameId: string) => void;
  /** Called when a game is deleted by another user in the workshop */
  onGameDeleted?: (gameId: string) => void;
}

/**
 * Hook to subscribe to real-time workshop events via SSE.
 * Automatically invalidates relevant queries when workshop settings change.
 *
 * Connect when entering workshop view, disconnect on unmount.
 */
export function useWorkshopEvents(options: UseWorkshopEventsOptions) {
  const {
    workshopId,
    enabled = true,
    onSettingsUpdate,
    onGameCreated,
    onGameUpdated,
    onGameDeleted,
  } = options;
  const { backendUser } = useAuth();
  const queryClient = useQueryClient();
  const { getAccessToken, isParticipant } = useAuth();
  const eventSourceRef = useRef<EventSource | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  // Use refs for callbacks to avoid reconnecting when they change
  const callbacksRef = useRef({
    onSettingsUpdate,
    onGameCreated,
    onGameUpdated,
    onGameDeleted,
  });
  // Update refs on each render
  callbacksRef.current = {
    onSettingsUpdate,
    onGameCreated,
    onGameUpdated,
    onGameDeleted,
  };
  const backendUserIdRef = useRef(backendUser?.id);
  backendUserIdRef.current = backendUser?.id;

  useEffect(() => {
    if (!workshopId || !enabled) {
      return;
    }

    let eventSource: EventSource | null = null;
    let cancelled = false;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let reconnectAttempt = 0;
    const MAX_RECONNECT_DELAY = 30_000; // 30 seconds max
    const BASE_RECONNECT_DELAY = 1_000; // 1 second initial

    const getReconnectDelay = () =>
      Math.min(
        BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttempt),
        MAX_RECONNECT_DELAY,
      );

    const scheduleReconnect = () => {
      if (cancelled) return;
      const delay = getReconnectDelay();
      reconnectAttempt++;
      uiLogger.debug("Scheduling SSE reconnect", {
        workshopId,
        attempt: reconnectAttempt,
        delayMs: delay,
      });
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        if (!cancelled) connect();
      }, delay);
    };

    const connect = async () => {
      const baseUrl = config.API_BASE_URL.replace(/\/$/, "");
      let url = `${baseUrl}/workshops/${workshopId}/events`;

      // For non-participant users, add token as query param (EventSource can't send headers)
      // Participants use cookie auth which is sent automatically
      if (!isParticipant) {
        const token = await getAccessToken();
        if (!token) {
          uiLogger.debug(
            "No token available for workshop SSE, skipping connection",
          );
          return;
        }
        url = `${url}?token=${encodeURIComponent(token)}`;
      }

      if (cancelled) return;

      uiLogger.debug("Connecting to workshop SSE", {
        workshopId,
        isParticipant,
      });

      eventSource = new EventSource(url, { withCredentials: true });
      eventSourceRef.current = eventSource;

      eventSource.addEventListener("connected", () => {
        uiLogger.info("Workshop SSE connected", { workshopId });
        reconnectAttempt = 0; // Reset backoff on successful connection
        setIsConnected(true);
      });

      eventSource.addEventListener("workshop_updated", () => {
        uiLogger.info("Workshop settings updated, refreshing data", {
          workshopId,
        });

        // Call the callback to refresh backend user (stored in AuthProvider state, not TanStack Query)
        callbacksRef.current.onSettingsUpdate?.();

        // Invalidate all queries affected by workshop settings changes:
        // - games: visibility settings (showPublicGames, showOtherParticipantsGames)
        // - currentUser: workshop settings like aiQualityTier are in user.role.workshop
        // - availableKeys: workshop API key changes affect available keys for games
        queryClient.invalidateQueries({ queryKey: queryKeys.games });
        queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
        queryClient.invalidateQueries({ queryKey: ["availableKeys"] });
      });

      // Helper to parse game event data and check if we triggered it
      const handleGameEvent = (
        event: MessageEvent,
        callback?: (gameId: string) => void,
      ) => {
        try {
          const data = JSON.parse(event.data) as GameEventData;
          // Ignore events triggered by ourselves
          const currentUserId = backendUserIdRef.current;
          if (currentUserId && data.triggeredBy === currentUserId) {
            uiLogger.debug("Ignoring own game event", { gameId: data.gameId });
            return;
          }
          callback?.(data.gameId);
        } catch (e) {
          uiLogger.warning("Failed to parse game event data", { error: e });
        }
      };

      eventSource.addEventListener("game_created", (event: MessageEvent) => {
        uiLogger.info("Game created in workshop", { workshopId });
        handleGameEvent(event, callbacksRef.current.onGameCreated);
      });

      eventSource.addEventListener("game_updated", (event: MessageEvent) => {
        uiLogger.info("Game updated in workshop", { workshopId });
        handleGameEvent(event, callbacksRef.current.onGameUpdated);
      });

      eventSource.addEventListener("game_deleted", (event: MessageEvent) => {
        uiLogger.info("Game deleted in workshop", { workshopId });
        handleGameEvent(event, callbacksRef.current.onGameDeleted);
      });

      eventSource.onerror = () => {
        uiLogger.warning("Workshop SSE error, reconnecting with backoff", {
          workshopId,
          attempt: reconnectAttempt,
        });
        setIsConnected(false);
        // Close the native EventSource to prevent its own auto-reconnect
        eventSource?.close();
        eventSource = null;
        eventSourceRef.current = null;
        // Reconnect manually with exponential backoff
        scheduleReconnect();
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (reconnectTimer) {
        clearTimeout(reconnectTimer);
      }
      if (eventSource) {
        uiLogger.debug("Closing workshop SSE connection", { workshopId });
        eventSource.close();
      }
      eventSourceRef.current = null;
      setIsConnected(false);
    };
  }, [workshopId, enabled, queryClient, getAccessToken, isParticipant]);

  return { isConnected };
}
