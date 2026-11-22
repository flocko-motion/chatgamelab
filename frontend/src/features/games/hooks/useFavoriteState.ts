import { useAddFavorite, useRemoveFavorite } from "@/api/hooks";
import type { GameView } from "@/api/types";

type GameRef = Pick<GameView, "id" | "isFavorite">;

/**
 * The favorite star of the game lists: each game says whether it is the
 * caller's favorite (isFavorite); toggling marks or unmarks it.
 */
export function useFavoriteState() {
  const addFavorite = useAddFavorite();
  const removeFavorite = useRemoveFavorite();

  const isFavorite = (game: GameRef) => game.isFavorite;

  const toggleFavorite = (game: GameRef) => {
    if (game.isFavorite) {
      removeFavorite.mutate(game.id);
    } else {
      addFavorite.mutate(game.id);
    }
  };

  return { isFavorite, toggleFavorite };
}
