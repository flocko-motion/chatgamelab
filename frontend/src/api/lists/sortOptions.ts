import type { SortDir } from "@/common/lib/listQuery";
import type { ListSortOption } from "./types";

/**
 * An order offered both ways, in the given order of directions (default:
 * descending first); `label` names each direction's i18n key.
 */
export function bothWays<S extends string>(
  by: S,
  label?: (dir: SortDir) => string,
  dirs: readonly [SortDir, SortDir] = ["desc", "asc"],
): ListSortOption<S>[] {
  return dirs.map((dir) => ({ by, dir, label: label?.(dir) }));
}
