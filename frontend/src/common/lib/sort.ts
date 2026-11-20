/**
 * Parse a combined "field-direction" sort value into its components.
 * Used by all game list views, invites list, workshops tab, etc.
 *
 * @example parseSortValue("modifiedAt-desc") // ["modifiedAt", "desc"]
 */
export function parseSortValue<T extends string = string>(
  sortValue: string,
): [field: T, direction: "asc" | "desc"] {
  const [field, dir] = sortValue.split("-") as [T, "asc" | "desc"];
  return [field, dir];
}
