import type { TFunction } from "i18next";
import type { GameListFilter } from "@/api/hooks";

/**
 * Why a game list came back empty, in one sentence built from what narrowed
 * it: "No game is one of your favorites, public and has “pirat” in its name."
 * The toggles are joined as the language joins a list (Intl.ListFormat).
 * Null when nothing narrowed the list: then there simply are no games.
 */
export function emptyListDescription(
  t: TFunction,
  language: string,
  only: readonly GameListFilter[],
  search: string,
): string | null {
  const conditions = new Intl.ListFormat(language, { type: "conjunction" }).format(
    only.map((o) => t(`allGames.empty.conditions.${o}`)),
  );
  const term = search.trim();
  if (only.length > 0 && term) return t("allGames.empty.isAndNamed", { conditions, search: term });
  if (only.length > 0) return t("allGames.empty.is", { conditions });
  if (term) return t("allGames.empty.named", { search: term });
  return null;
}
