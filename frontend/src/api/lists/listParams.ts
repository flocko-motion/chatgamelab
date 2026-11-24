import { listOffset, type ListQuery } from "@/common/lib/listQuery";
import type { ListParams, ListPreset } from "./types";

/** The single filter of a list that offers none. */
export const NO_FILTER = "all";

/**
 * The parameters a list's query asks its endpoint for: the preset's base
 * filters, what the picked filter applies, the toggles that are on, the
 * multi-filter's options picked, the search, the order and the page.
 */
export function listParams<F extends string, S extends string, O extends string>(
  query: ListQuery<F, S, O>,
  preset: Pick<ListPreset<F, S, O>, "base" | "filters">,
): ListParams {
  const picked = preset.filters?.find((f) => f.value === query.filter);
  const applied: readonly string[] = picked ? (picked.apply ?? [picked.value]) : [];
  const filter = [...(preset.base ?? []), ...applied, ...query.only, ...query.picked];
  return {
    filter: filter.length > 0 ? filter : undefined,
    search: query.search || undefined,
    sortBy: query.sortBy,
    sortDir: query.sortDir,
    limit: query.pageSize,
    offset: listOffset(query) || undefined,
  };
}
