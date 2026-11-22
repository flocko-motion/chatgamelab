import { platformFilter, type OrganisationKeyListSort } from "@/api/hooks";
import { bothWays } from "@/api/lists";
import type { ListPreset } from "@/api/types";
import type { PlatformInfo } from "@/api/types";

/**
 * The keys given to the organisation: search by name or owner, a filter per
 * platform, headers that sort by name, owner and platform.
 */
export type OrgKeyColumnKey = "name" | "owner" | "platform" | "actions";

export const ORG_KEY_COLUMNS: OrgKeyColumnKey[] = ["name", "owner", "platform", "actions"];

export type OrgKeyListPreset = ListPreset<string, OrganisationKeyListSort>;

/**
 * The preset for the platforms there are: every key, then one filter per
 * platform, labelled with its name. Built from data, so the caller memoizes it.
 */
export function orgKeyListPreset(platforms: readonly Pick<PlatformInfo, "id" | "name">[]): OrgKeyListPreset {
  return {
    filters: [
      { value: "all", label: "myOrganization.apiKeys.allPlatforms", apply: [] },
      ...platforms.map((p) => ({ value: platformFilter(p.id), text: p.name })),
    ],
    // Headers only: the list has no sort menu.
    sorts: [
      ...bothWays("name", undefined, ["asc", "desc"]),
      ...bothWays("owner", undefined, ["asc", "desc"]),
      ...bothWays("platform", undefined, ["asc", "desc"]),
    ],
    defaultSort: { by: "name", dir: "asc" },
  };
}
