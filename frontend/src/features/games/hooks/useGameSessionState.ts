import { useCallback } from "react";
import { useGameSessionMap } from "@/api/hooks";
import type { Game, SessionView } from "@/api/generated/api.gen";

export interface GameSessionState {
  hasSession: boolean;
  session: SessionView | undefined;
}

/**
 * Shared hook for looking up a game's active session from the session map.
 * Used by the game lists (GamesTable).
 */
export function useGameSessionState() {
  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();

  const getSessionState = useCallback(
    (game: Pick<Game, "id">): GameSessionState => {
      if (!game.id) return { hasSession: false, session: undefined };
      const session = sessionMap.get(game.id);
      return { hasSession: !!session, session };
    },
    [sessionMap],
  );

  return { sessionMap, sessionsLoading, getSessionState };
}
