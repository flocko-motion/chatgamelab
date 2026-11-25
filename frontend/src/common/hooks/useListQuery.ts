import { useCallback, useMemo, useState } from "react";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { useDebouncedCallback } from "@mantine/hooks";
import {
  DEFAULT_PAGE_SIZE,
  PAGE_SIZES,
  type ListQuery,
  type ListSearch,
  type SortDir,
} from "@/common/lib/listQuery";

export interface UseListQueryOptions<F extends string, S extends string, O extends string> {
  /** The filters the list knows; the first is the default. */
  filters: readonly F[];
  /** The toggles the list knows, each off by default. */
  only?: readonly O[];
  /** The multi-filter's options, none picked by default. */
  picks?: readonly O[];
  /** The orders the list knows. */
  sorts: readonly S[];
  defaultSort: { by: S; dir: SortDir };
  defaultPageSize?: number;
  /** Keep the query in the URL (default true), so back and links keep it. */
  inUrl?: boolean;
}

/** The query of a list and the ways to change it. */
export interface ListQueryControls<
  F extends string = string,
  S extends string = string,
  O extends string = string,
> {
  /** The query to fetch with; its search is debounced. */
  query: ListQuery<F, S, O>;
  /** What is in the search box right now, before the debounce. */
  searchInput: string;
  setSearch: (search: string) => void;
  setFilter: (filter: F) => void;
  /** Switches a toggle on or off. */
  toggleOnly(only: O): void;
  /** Picks a multi-filter option, or unpicks it. */
  togglePick(pick: O): void;
  /** Unpicks every multi-filter option. */
  clearPicks(): void;
  /** Picks exactly these multi-filter options; none is the same as clearPicks. */
  setPicks(picks: readonly O[]): void;
  setSort: (by: S, dir: SortDir) => void;
  /** Sorts by a column, or flips the direction if it is sorted by it already. */
  toggleSort: (by: S) => void;
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
}

const SEARCH_DEBOUNCE_MS = 300;

const NO_TOGGLES: readonly never[] = [];

/**
 * Holds a server-side list's query: filter, toggles, picks, search, order and page.
 * Changing the filter, a toggle, a pick, search, order or page size goes back to
 * the first page. Values
 * the list doesn't know (an old link, a hand-edited URL) fall back to the
 * defaults, so the server is never asked for them.
 */
export function useListQuery<F extends string, S extends string, O extends string = never>(
  options: UseListQueryOptions<F, S, O>,
): ListQueryControls<F, S, O> {
  const {
    filters,
    only: toggles = NO_TOGGLES,
    picks = NO_TOGGLES,
    sorts,
    defaultSort,
    inUrl = true,
  } = options;
  const defaultPageSize = options.defaultPageSize ?? DEFAULT_PAGE_SIZE;

  const [local, setLocal] = useState<ListSearch>({});
  const urlSearch = useSearch({ strict: false }) as ListSearch;
  const navigate = useNavigate();
  const raw = inUrl ? urlSearch : local;

  const update = useCallback(
    (patch: ListSearch, replace = false) => {
      if (!inUrl) {
        setLocal((prev) => ({ ...prev, ...patch }));
        return;
      }
      navigate({
        to: ".",
        search: ((prev: Record<string, unknown>) => ({ ...prev, ...patch })) as never,
        replace,
      });
    },
    [inUrl, navigate],
  );

  const filter = filters.includes(raw.filter as F) ? (raw.filter as F) : filters[0];
  const onlyKey = raw.only ?? "";
  const only = useMemo(
    () => toggles.filter((o) => onlyKey.split(",").includes(o)),
    [toggles, onlyKey],
  );
  const pickKey = raw.pick ?? "";
  const picked = useMemo(
    () => picks.filter((p) => pickKey.split(",").includes(p)),
    [picks, pickKey],
  );
  const sortBy = sorts.includes(raw.sort as S) ? (raw.sort as S) : defaultSort.by;
  const sortDir = raw.dir ?? (raw.sort ? "asc" : defaultSort.dir);
  const pageSize = (PAGE_SIZES as readonly number[]).includes(raw.size ?? 0)
    ? (raw.size as number)
    : defaultPageSize;
  const page = raw.page ?? 1;
  const search = raw.q ?? "";

  // The search box is typed into locally and reaches the query after a pause.
  // A change of the query from elsewhere (back, a link) refills the box.
  const [searchInput, setSearchInput] = useState(search);
  const [shownSearch, setShownSearch] = useState(search);
  if (search !== shownSearch) {
    setShownSearch(search);
    setSearchInput(search);
  }
  const pushSearch = useDebouncedCallback(
    (value: string) => update({ q: value || undefined, page: undefined }, true),
    SEARCH_DEBOUNCE_MS,
  );
  const setSearch = useCallback(
    (value: string) => {
      setSearchInput(value);
      pushSearch(value);
    },
    [pushSearch],
  );

  const query = useMemo(
    () => ({ filter, only, picked, search, sortBy, sortDir, page, pageSize }),
    [filter, only, picked, search, sortBy, sortDir, page, pageSize],
  );

  // Defaults stay out of the URL, so a plain URL is the default list.
  const setFilter = useCallback(
    (f: F) => update({ filter: f === filters[0] ? undefined : f, page: undefined }),
    [filters, update],
  );
  const toggleOnly = useCallback(
    (o: O) => {
      const next = only.includes(o) ? only.filter((x) => x !== o) : [...only, o];
      update({ only: next.length > 0 ? next.join(",") : undefined, page: undefined });
    },
    [only, update],
  );
  const togglePick = useCallback(
    (p: O) => {
      const next = picked.includes(p) ? picked.filter((x) => x !== p) : [...picked, p];
      update({ pick: next.length > 0 ? next.join(",") : undefined, page: undefined });
    },
    [picked, update],
  );
  const clearPicks = useCallback(
    () => update({ pick: undefined, page: undefined }),
    [update],
  );
  const setPicks = useCallback(
    (picks: readonly O[]) =>
      update({ pick: picks.length > 0 ? picks.join(",") : undefined, page: undefined }),
    [update],
  );
  const setSort = useCallback(
    (by: S, dir: SortDir) => {
      const isDefault = by === defaultSort.by && dir === defaultSort.dir;
      update({
        sort: isDefault ? undefined : by,
        dir: isDefault ? undefined : dir,
        page: undefined,
      });
    },
    [defaultSort.by, defaultSort.dir, update],
  );
  const toggleSort = useCallback(
    (by: S) => {
      if (by === sortBy) setSort(by, sortDir === "asc" ? "desc" : "asc");
      else setSort(by, by === defaultSort.by ? defaultSort.dir : "asc");
    },
    [sortBy, sortDir, setSort, defaultSort.by, defaultSort.dir],
  );
  const setPage = useCallback(
    (p: number) => update({ page: p > 1 ? p : undefined }),
    [update],
  );
  const setPageSize = useCallback(
    (size: number) =>
      update({ size: size === defaultPageSize ? undefined : size, page: undefined }),
    [defaultPageSize, update],
  );

  return {
    query,
    searchInput,
    setSearch,
    setFilter,
    toggleOnly,
    togglePick,
    clearPicks,
    setPicks,
    setSort,
    toggleSort,
    setPage,
    setPageSize,
  };
}
