import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { useAuthenticatedApi } from "../useAuthenticatedApi";
import { uiLogger } from "@/config/logger";
import type { ObjGame } from "../generated";

/**
 * Hook that provides functions to update the games cache without full refetch.
 * Used for SSE events to add/update/remove single games.
 *
 * For participants (cookie auth), falls back to invalidating the games query
 * since they can't use the token-based API client.
 */
export function useGamesCacheUpdater() {
  const queryClient = useQueryClient();
  const api = useAuthenticatedApi();

  /**
   * Invalidate games query as fallback when we can't fetch individual games
   */
  const invalidateGames = useCallback(() => {
    uiLogger.debug("Invalidating games query (fallback)");
    queryClient.invalidateQueries({ queryKey: queryKeys.games });
  }, [queryClient]);

  /**
   * Fetch a single game and add it to all games list caches.
   * Falls back to full invalidation if API client unavailable.
   */
  const addGameToCache = useCallback(
    async (gameId: string) => {
      uiLogger.debug("addGameToCache called", { gameId, hasApi: !!api });
      if (!api) {
        // API not available - fall back to invalidating games
        uiLogger.debug("No API client, falling back to invalidation");
        invalidateGames();
        return;
      }

      try {
        const response = await api.games.gamesDetail(gameId);
        const newGame = response.data;

        // Update all games list queries
        queryClient.setQueriesData<ObjGame[]>(
          { queryKey: queryKeys.games },
          (oldData) => {
            if (!oldData) return [newGame];
            // Check if game already exists (avoid duplicates)
            if (oldData.some((g) => g.id === gameId)) {
              return oldData.map((g) => (g.id === gameId ? newGame : g));
            }
            return [newGame, ...oldData];
          },
        );

        uiLogger.debug("Added game to cache", { gameId });
      } catch (error) {
        uiLogger.warning(
          "Failed to fetch game for cache update, invalidating",
          {
            gameId,
            error,
          },
        );
        // Fall back to invalidation on error
        invalidateGames();
      }
    },
    [api, queryClient, invalidateGames],
  );

  /**
   * Fetch a single game and update it in all games list caches.
   * Falls back to full invalidation for participants.
   */
  const updateGameInCache = useCallback(
    async (gameId: string) => {
      if (!api) {
        // Participants use cookie auth - fall back to invalidating games
        invalidateGames();
        return;
      }

      try {
        const response = await api.games.gamesDetail(gameId);
        const updatedGame = response.data;

        // Update all games list queries
        queryClient.setQueriesData<ObjGame[]>(
          { queryKey: queryKeys.games },
          (oldData) => {
            if (!oldData) return oldData;
            return oldData.map((g) => (g.id === gameId ? updatedGame : g));
          },
        );

        // Also update single game query if it exists
        queryClient.setQueryData(queryKeys.game(gameId), updatedGame);

        uiLogger.debug("Updated game in cache", { gameId });
      } catch (error) {
        uiLogger.warning(
          "Failed to fetch game for cache update, invalidating",
          {
            gameId,
            error,
          },
        );
        // Fall back to invalidation on error
        invalidateGames();
      }
    },
    [api, queryClient, invalidateGames],
  );

  /**
   * Remove a game from all games list caches
   */
  const removeGameFromCache = useCallback(
    (gameId: string) => {
      // Update all games list queries
      queryClient.setQueriesData<ObjGame[]>(
        { queryKey: queryKeys.games },
        (oldData) => {
          if (!oldData) return oldData;
          return oldData.filter((g) => g.id !== gameId);
        },
      );

      // Remove single game query if it exists
      queryClient.removeQueries({ queryKey: queryKeys.game(gameId) });

      uiLogger.debug("Removed game from cache", { gameId });
    },
    [queryClient],
  );

  return {
    addGameToCache,
    updateGameInCache,
    removeGameFromCache,
  };
}
