import { useEffect } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { pageCount, type ListPage } from "@/common/lib/listQuery";
import type { ListQueryControls } from "./useListQuery";

/** What a list table shows: the current page and its loading state. */
export interface ServerList<T> {
  items: T[];
  total: number;
  /** Nothing to show yet. */
  isInitialLoading: boolean;
  /** A page is shown while the next one loads. */
  isRefetching: boolean;
  error: unknown;
}

/**
 * Turns a query for a page into what a list table shows. When the list
 * shrinks under the current page (games deleted elsewhere), it moves to the
 * last page that still exists.
 */
export function useServerList<T>(
  controls: Pick<ListQueryControls, "setPage"> & {
    query: Pick<ListQueryControls["query"], "page" | "pageSize">;
  },
  result: UseQueryResult<ListPage<T>, unknown>,
): ServerList<T> {
  const { data, isLoading, isFetching, isPlaceholderData, error } = result;
  const total = data?.total ?? 0;
  const { page, pageSize } = controls.query;
  const { setPage } = controls;

  useEffect(() => {
    if (!data || isPlaceholderData) return;
    const last = pageCount(data.total, pageSize);
    if (page > last) setPage(last);
  }, [data, isPlaceholderData, page, pageSize, setPage]);

  return {
    items: data?.items ?? [],
    total,
    isInitialLoading: !data && isLoading,
    isRefetching: !!data && isFetching,
    error,
  };
}
