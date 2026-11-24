import { useGenres } from "@/api/hooks";
import type { GameField } from "@/api/types";

/**
 * Whether the editor shows a game field for a genre: the engine says which
 * fields each genre reads. Until the genres are known, and for a genre the
 * engine does not list, every field shows, so nothing a game holds is hidden.
 */
export function useGenreFields(genre: string): (field: GameField) => boolean {
  const { data: genres } = useGenres();
  const fields = genres?.find((g) => g.id === genre)?.fields;
  return (field) => !fields || fields.includes(field);
}
