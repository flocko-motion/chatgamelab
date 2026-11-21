import { api } from "../client/api";
import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../queryKeys";
import { uiLogger } from "@/config/logger";

/**
 * Keeps the games cache current on SSE events (a game created, changed or
 * deleted elsewhere). Game lists are server-side pages: where a game lands,
 * and the total, only the server knows, so the lists are refetched (one page
 * each) instead of patched.
 */
export function useGamesCacheUpdater() {
  const queryClient = useQueryClient();

  const refetchLists = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.gameLists });
  }, [queryClient]);

  const addGameToCache = useCallback(
    (gameId: string) => {
      uiLogger.debug("Game created, refetching game lists", { gameId });
      refetchLists();
    },
    [refetchLists],
  );

  const updateGameInCache = useCallback(
    async (gameId: string) => {
      refetchLists();
      if (!queryClient.getQueryData(queryKeys.game(gameId))) return;
      try {
        const response = await api.getApiGamesId(gameId);
        queryClient.setQueryData(queryKeys.game(gameId), response.data);
      } catch (error) {
        uiLogger.warning("Failed to refetch updated game", { gameId, error });
        queryClient.invalidateQueries({ queryKey: queryKeys.game(gameId) });
      }
    },
    [queryClient, refetchLists],
  );

  const removeGameFromCache = useCallback(
    (gameId: string) => {
      refetchLists();
      queryClient.removeQueries({ queryKey: queryKeys.game(gameId) });
    },
    [queryClient, refetchLists],
  );

  return {
    refetchLists,
    addGameToCache,
    updateGameInCache,
    removeGameFromCache,
  };
}
