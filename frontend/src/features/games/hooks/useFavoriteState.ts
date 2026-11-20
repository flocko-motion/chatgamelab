import { useMemo } from "react";
import {
  useFavoriteGames,
  useAddFavorite,
  useRemoveFavorite,
} from "@/api/hooks";
import type { Game } from "@/api/generated/api.gen";

/**
 * Shared hook for game favorites state and toggle logic.
 * Used by AllGames, MyGames, and any other game list view.
 */
export function useFavoriteState() {
  const { data: favoriteGames } = useFavoriteGames();
  const addFavorite = useAddFavorite();
  const removeFavorite = useRemoveFavorite();

  const favoriteGameIds = useMemo(
    () => new Set(favoriteGames?.map((g) => g.id) ?? []),
    [favoriteGames],
  );

  const isFavorite = (game: Game) =>
    game.id ? favoriteGameIds.has(game.id) : false;

  const toggleFavorite = (game: Game) => {
    if (!game.id) return;
    if (isFavorite(game)) {
      removeFavorite.mutate(game.id);
    } else {
      addFavorite.mutate(game.id);
    }
  };

  return { favoriteGameIds, isFavorite, toggleFavorite };
}
