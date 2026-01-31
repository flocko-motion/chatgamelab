import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { config } from "@/config/env";
import { queryKeys } from "../queryKeys";
import { uiLogger } from "@/config/logger";
import { useAuth } from "@/providers/AuthProvider";

interface UseWorkshopEventsOptions {
  workshopId: string | undefined;
  enabled?: boolean;
  /** Called when workshop settings are updated - use to refresh backend user */
  onSettingsUpdate?: () => void;
}

/**
 * Hook to subscribe to real-time workshop events via SSE.
 * Automatically invalidates relevant queries when workshop settings change.
 *
 * Connect when entering workshop view, disconnect on unmount.
 */
export function useWorkshopEvents(options: UseWorkshopEventsOptions) {
  const { workshopId, enabled = true, onSettingsUpdate } = options;
  const queryClient = useQueryClient();
  const { getAccessToken, isParticipant } = useAuth();
  const eventSourceRef = useRef<EventSource | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    if (!workshopId || !enabled) {
      return;
    }

    let eventSource: EventSource | null = null;
    let cancelled = false;

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
        setIsConnected(true);
      });

      eventSource.addEventListener("workshop_updated", () => {
        uiLogger.info("Workshop settings updated, refreshing data", {
          workshopId,
        });

        // Call the callback to refresh backend user (stored in AuthProvider state, not TanStack Query)
        onSettingsUpdate?.();

        // Invalidate all queries affected by workshop settings changes:
        // - games: visibility settings (showPublicGames, showOtherParticipantsGames)
        // - currentUser: workshop settings like showAiModelSelector are in user.role.workshop
        // - availableKeys: workshop API key changes affect available keys for games
        queryClient.invalidateQueries({ queryKey: queryKeys.games });
        queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
        queryClient.invalidateQueries({ queryKey: ["availableKeys"] });
      });

      eventSource.onerror = () => {
        uiLogger.warning("Workshop SSE error, will auto-reconnect", {
          workshopId,
        });
        setIsConnected(false);
        // EventSource will auto-reconnect
      };
    };

    connect();

    return () => {
      cancelled = true;
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
