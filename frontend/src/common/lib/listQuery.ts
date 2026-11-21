/**
 * The query a server-side list is asked with: which filter, what to search
 * for, in which order, and which page. The server does the filtering, sorting
 * and paging; the client only holds this query and shows the page it gets.
 */

export type SortDir = "asc" | "desc";

export interface ListQuery<
  F extends string = string,
  S extends string = string,
  O extends string = string,
> {
  filter: F;
  /** Toggles that each narrow the list on top of the filter, e.g. favorites. */
  only: O[];
  search: string;
  sortBy: S;
  sortDir: SortDir;
  /** 1-based */
  page: number;
  pageSize: number;
}

/** A page of a list, and how many items match across all pages. */
export interface ListPage<T> {
  items: T[];
  total: number;
}

/** The page sizes a list offers. */
export const PAGE_SIZES = [10, 25, 50, 100] as const;

export const DEFAULT_PAGE_SIZE = 25;

/** How many items to skip for the query's page. */
export function listOffset(query: Pick<ListQuery, "page" | "pageSize">): number {
  return (query.page - 1) * query.pageSize;
}

/** How many pages a list of `total` items has (at least one). */
export function pageCount(total: number, pageSize: number): number {
  return Math.max(1, Math.ceil(total / pageSize));
}

/**
 * The list query as it sits in the URL. Every key is optional: a missing key
 * means the list's default, so a plain URL shows the default list.
 */
export interface ListSearch {
  filter?: string;
  /** The toggles that are on, comma-separated. */
  only?: string;
  q?: string;
  sort?: string;
  dir?: SortDir;
  page?: number;
  size?: number;
}

/**
 * Reads the list keys from a route's search params. Routes that show a list
 * use it in `validateSearch`, so the keys survive navigation. Values are only
 * checked for their shape here; whether a filter or order exists is up to the
 * list (`useListQuery` falls back to its defaults).
 */
export function validateListSearch(search: Record<string, unknown>): ListSearch {
  const text = (v: unknown) => (typeof v === "string" && v !== "" ? v : undefined);
  const positive = (v: unknown) => {
    const n = typeof v === "number" ? v : typeof v === "string" ? Number(v) : NaN;
    return Number.isInteger(n) && n > 0 ? n : undefined;
  };
  return {
    filter: text(search.filter),
    only: text(search.only),
    q: text(search.q),
    sort: text(search.sort),
    dir: search.dir === "asc" || search.dir === "desc" ? search.dir : undefined,
    page: positive(search.page),
    size: positive(search.size),
  };
}
