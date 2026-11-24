import { useMemo } from "react";
import { keepPreviousData, useQuery, type QueryKey } from "@tanstack/react-query";
import { useListQuery, type UseListQueryOptions } from "@/common/hooks/useListQuery";
import { useServerList } from "@/common/hooks/useServerList";
import type { ListPage } from "@/common/lib/listQuery";
import { listParams, NO_FILTER } from "./listParams";
import type { List, ListParams, ListPreset } from "./types";

/** An entity's list endpoint: it takes the list parameters and returns a page. */
export interface ListSource<T> {
  key: (params: ListParams) => QueryKey;
  fetch: (params: ListParams) => Promise<ListPage<T>>;
}

const NO_FILTERS = [NO_FILTER] as const;
const NO_TOGGLES: readonly never[] = [];

/** What `useListQuery` takes, from a preset; cached per preset so it stays the same. */
const queryOptions = new WeakMap<object, UseListQueryOptions<string, string, string>>();
function queryOptionsOf<F extends string, S extends string, O extends string>(
  preset: ListPreset<F, S, O>,
): UseListQueryOptions<F, S, O> {
  let options = queryOptions.get(preset);
  if (!options) {
    options = {
      filters: preset.filters?.map((f) => f.value) ?? NO_FILTERS,
      only: preset.toggles?.map((o) => o.value) ?? NO_TOGGLES,
      picks: preset.multiFilter?.options.map((o) => o.value) ?? NO_TOGGLES,
      sorts: [...new Set(preset.sorts.map((s) => s.by))],
      defaultSort: preset.defaultSort,
      defaultPageSize: preset.defaultPageSize,
      inUrl: preset.inUrl,
    };
    queryOptions.set(preset, options);
  }
  return options as unknown as UseListQueryOptions<F, S, O>;
}

/**
 * Builds an entity's `List<T>`: the query (in the URL unless the preset says
 * otherwise), the page the endpoint returns for it, and the state a table
 * shows. Entity hooks in `api/` call it; features take the list from them.
 */
export function useEntityList<T, F extends string, S extends string, O extends string = never>(
  preset: ListPreset<F, S, O>,
  source: ListSource<T>,
  options: { enabled?: boolean } = {},
): List<T, F, S, O> {
  const controls = useListQuery<F, S, O>(queryOptionsOf(preset));
  const { query } = controls;
  const params = useMemo(() => listParams(query, preset), [query, preset]);

  const result = useQuery<ListPage<T>, Error>({
    queryKey: source.key(params),
    queryFn: () => source.fetch(params),
    placeholderData: keepPreviousData,
    enabled: options.enabled ?? true,
    refetchInterval: preset.refetchInterval,
  });

  const list = useServerList<T>(controls, result);
  return { ...controls, ...list, preset };
}
