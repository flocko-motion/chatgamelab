import { genreFilter, type GameListFilter, type GameListSort } from "@/api/hooks";
import { bothWays } from "@/api/lists";
import type { ListPreset } from "@/api/types";

/**
 * The game lists the app shows. A preset says what a list offers: its
 * filters, orders and columns, and whether games can be created from it.
 * Who may do what with a game is not a preset's business (useGamesTable).
 */
export type GameListPreset = "all" | "workshop";

export type GameColumnKey =
  | "favorite"
  | "name"
  | "creator"
  | "workshop"
  | "badges"
  | "playCount"
  | "visibility"
  | "date"
  | "actions";

export interface GameListPresetConfig {
  /**
   * What the list offers: filters, toggles (the favorite column's header
   * star is a shortcut to "favorites") and orders.
   */
  list: ListPreset<string, GameListSort, GameListFilter>;
  columns: GameColumnKey[];
  /** The list has create and import buttons. */
  canCreate: boolean;
  /** A created game (new, copied, imported) opens afterwards. */
  openCreatedGame: boolean;
  empty: { titleKey: string; descriptionKey: string; ns?: string };
}

/**
 * A game list that can be narrowed to genres, picked together: one option per
 * genre the engine plays. Built from the genres, so the caller keeps it
 * stable (useMemo).
 */
export function withGenreFilter(
  list: GameListPresetConfig["list"],
  genres: readonly string[],
): GameListPresetConfig["list"] {
  if (genres.length < 2) return list;
  return {
    ...list,
    multiFilter: {
      label: "allGames.filters.genre",
      options: genres.map((genre) => ({ value: genreFilter(genre), label: `games.genres.${genre}` })),
    },
  };
}

const both = (by: GameListSort) => bothWays(by, (dir) => `games.sort.${by}-${dir}`);

export const GAME_LIST_PRESETS: Record<GameListPreset, GameListPresetConfig> = {
  all: {
    list: {
      toggles: [
        { value: "favorites", label: "allGames.filters.favorites" },
        { value: "own", label: "allGames.filters.own" },
        { value: "sponsored", label: "allGames.filters.sponsored" },
        { value: "public", label: "allGames.filters.public" },
      ],
      sorts: [
        ...both("modifiedAt"),
        ...both("createdAt"),
        ...both("name"),
        ...both("creator"),
        ...both("playCount"),
        ...both("visibility"),
      ],
      defaultSort: { by: "modifiedAt", dir: "desc" },
    },
    columns: ["favorite", "name", "creator", "workshop", "playCount", "visibility", "date", "actions"],
    canCreate: true,
    openCreatedGame: true,
    empty: { titleKey: "allGames.empty.title", descriptionKey: "allGames.empty.description" },
  },
  workshop: {
    // In workshop mode the server lists the workshop's world; workshopId on
    // each game tells the workshop's games from released public ones.
    list: {
      filters: [
        { value: "all", label: "myWorkshop:filters.all", apply: [] },
        { value: "own", label: "myWorkshop:filters.mine", apply: ["own"] },
        { value: "public", label: "myWorkshop:filters.public", apply: ["public"] },
      ],
      sorts: [...both("modifiedAt"), ...both("createdAt"), ...both("name"), ...both("creator")],
      defaultSort: { by: "modifiedAt", dir: "desc" },
    },
    columns: ["name", "creator", "badges", "date", "actions"],
    canCreate: true,
    openCreatedGame: false,
    empty: { titleKey: "empty.title", descriptionKey: "empty.description", ns: "myWorkshop" },
  },
};
