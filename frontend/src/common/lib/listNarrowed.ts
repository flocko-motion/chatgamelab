import type { List } from "@/api/lists";

/**
 * Whether a list is narrowed by what the user picked: a search, a toggle, or
 * a filter other than the default (the preset's first). An empty narrowed list
 * means nothing matches; an empty plain one means there is nothing yet.
 */
export function isListNarrowed<F extends string, S extends string, O extends string>(
  list: Pick<List<unknown, F, S, O>, "query" | "preset">,
): boolean {
  const { query, preset } = list;
  const defaultFilter: string | undefined = preset.filters?.[0]?.value;
  return (
    query.search.trim() !== "" ||
    query.only.length > 0 ||
    (defaultFilter !== undefined && query.filter !== defaultFilter)
  );
}
