import type { GameListFilter, GameListSort } from "@/api/hooks";
import type { SortDir } from "@/common/lib/listQuery";

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

/**
 * A filter the user picks. It asks the server for `apply` (all of them, on
 * top of the preset's base filters); an empty `apply` adds nothing.
 */
export interface GameListFilterOption {
  value: string;
  labelKey: string;
  apply: GameListFilter[];
}

/** A filter that is on or off and combines with the others (and the filter). */
export interface GameListToggleOption {
  value: GameListFilter;
  labelKey: string;
}

export interface GameListPresetConfig {
  /** Always asked for, whatever the user picks. */
  baseFilters: GameListFilter[];
  /** The first is the default. */
  filters: GameListFilterOption[];
  /** Shown in the toolbar; the favorite column's header star is a shortcut to "favorites". */
  toggles: GameListToggleOption[];
  sorts: { by: GameListSort; dir: SortDir }[];
  defaultSort: { by: GameListSort; dir: SortDir };
  columns: GameColumnKey[];
  /** The list has create and import buttons. */
  canCreate: boolean;
  /** A created game (new, copied, imported) opens afterwards. */
  openCreatedGame: boolean;
  empty: { titleKey: string; descriptionKey: string; ns?: string };
}

const both = (by: GameListSort) =>
  [
    { by, dir: "desc" as const },
    { by, dir: "asc" as const },
  ];

export const GAME_LIST_PRESETS: Record<GameListPreset, GameListPresetConfig> = {
  all: {
    baseFilters: [],
    filters: [{ value: "all", labelKey: "allGames.filters.all", apply: [] }],
    toggles: [
      { value: "favorites", labelKey: "allGames.filters.favorites" },
      { value: "own", labelKey: "allGames.filters.own" },
      { value: "sponsored", labelKey: "allGames.filters.sponsored" },
      { value: "public", labelKey: "allGames.filters.public" },
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
    columns: ["favorite", "name", "creator", "workshop", "playCount", "visibility", "date", "actions"],
    canCreate: true,
    openCreatedGame: true,
    empty: { titleKey: "allGames.empty.title", descriptionKey: "allGames.empty.description" },
  },
  workshop: {
    // In workshop mode the server lists the workshop's world; workshopId on
    // each game tells the workshop's games from released public ones.
    baseFilters: [],
    filters: [
      { value: "all", labelKey: "filters.all", apply: [] },
      { value: "own", labelKey: "filters.mine", apply: ["own"] },
      { value: "public", labelKey: "filters.public", apply: ["public"] },
    ],
    sorts: [...both("modifiedAt"), ...both("createdAt"), ...both("name"), ...both("creator")],
    defaultSort: { by: "modifiedAt", dir: "desc" },
    toggles: [],
    columns: ["name", "creator", "badges", "date", "actions"],
    canCreate: true,
    openCreatedGame: false,
    empty: { titleKey: "empty.title", descriptionKey: "empty.description", ns: "myWorkshop" },
  },
};

/** The namespace of a preset's filter labels. */
export function presetFilterNamespace(preset: GameListPreset): string {
  return preset === "workshop" ? "myWorkshop" : "common";
}
