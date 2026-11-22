import type { ListQueryControls } from "@/common/hooks/useListQuery";
import type { ServerList } from "@/common/hooks/useServerList";
import type { SortDir } from "@/common/lib/listQuery";

/**
 * A list of records as the web app shows it: the current page, how many match
 * in all, the loading state, the query (filter, toggles, search, order, page)
 * and the ways to change it. Entity hooks in `api/` return one; `ListTable`
 * renders it.
 */
export type List<
  T,
  F extends string = string,
  S extends string = string,
  O extends string = string,
> = ServerList<T> &
  ListQueryControls<F, S, O> & {
    /** What the list offers: the toolbar's filters, toggles and sort menu. */
    preset: ListPreset<F, S, O>;
  };

/** A filter or toggle a list offers, and how it is labelled. */
export interface ListOption<V extends string> {
  value: V;
  /** The label's i18n key; a key in another namespace is written `ns:key`. */
  label?: string;
  /** A label that is data, shown as it is (e.g. a platform's name). */
  text?: string;
}

/** A filter: one at a time; it asks the server for `apply` (default: itself). */
export interface ListFilterOption<F extends string> extends ListOption<F> {
  /** The server filters it asks for; `[]` asks for nothing (e.g. "all"). */
  apply?: readonly string[];
}

/** An order the list knows; with a label it is also offered in the sort menu. */
export interface ListSortOption<S extends string> {
  by: S;
  dir: SortDir;
  /** The label's i18n key (`ns:key` for another namespace); none: headers only. */
  label?: string;
}

/**
 * What a list offers, in one place: its filters (one at a time, the first is
 * the default), its toggles (each on or off, combined), its orders and where
 * its query lives. A preset is declared once (module level, or memoized when
 * built from data), so it stays the same between renders.
 */
export interface ListPreset<F extends string, S extends string, O extends string = never> {
  /** None: one plain list. More than one: the toolbar offers them. */
  filters?: readonly ListFilterOption<F>[];
  toggles?: readonly ListOption<O>[];
  sorts: readonly ListSortOption<S>[];
  defaultSort: { by: S; dir: SortDir };
  /** Server filters always asked for, whatever the user picks. */
  base?: readonly string[];
  /** Keep the query in the URL (default true), so back and links keep it. */
  inUrl?: boolean;
  defaultPageSize?: number;
  /** Refresh the page this often (ms), e.g. for a list that changes elsewhere. */
  refetchInterval?: number;
}

/**
 * What every list endpoint takes: filters (repeatable, combined), search,
 * order and page. A part left out is the endpoint's default.
 */
export interface ListParams {
  filter?: string[];
  search?: string;
  sortBy?: string;
  sortDir?: SortDir;
  limit?: number;
  offset?: number;
}
