import { useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";
import type { ObjGame } from "@/api/generated";

/**
 * Shared hook for starting a game.
 * Used by AllGames, MyGames, and MyWorkshop.
 */
export function useGameNavigation() {
  const navigate = useNavigate();

  const playGame = useCallback(
    (game: ObjGame) => {
      if (game.id) {
        navigate({ to: "/games/$gameId/play", params: { gameId: game.id } });
      }
    },
    [navigate],
  );

  return { playGame };
}
