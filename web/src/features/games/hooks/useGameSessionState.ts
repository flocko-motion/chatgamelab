import { useCallback } from "react";
import { useGameSessionMap } from "@/api/hooks";
import type { ObjGame, DbUserSessionWithGame } from "@/api/generated";

export interface GameSessionState {
  hasSession: boolean;
  session: DbUserSessionWithGame | undefined;
}

/**
 * Shared hook for looking up a game's active session from the session map.
 * Used by AllGames, MyGames, and MyWorkshop.
 */
export function useGameSessionState() {
  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();

  const getSessionState = useCallback(
    (game: ObjGame): GameSessionState => {
      if (!game.id) return { hasSession: false, session: undefined };
      const session = sessionMap.get(game.id);
      return { hasSession: !!session, session };
    },
    [sessionMap],
  );

  return { sessionMap, sessionsLoading, getSessionState };
}
