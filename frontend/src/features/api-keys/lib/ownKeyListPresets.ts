import { platformFilter, type OwnKeyListSort } from "@/api/hooks";
import { bothWays } from "@/api/lists";
import type { ListPreset } from "@/api/types";

/** A platform the key list knows: one that takes keys, or one with stored keys only. */
export interface KeyPlatform {
  id: string;
  name: string;
}

export type OwnKeyListPreset = ListPreset<string, OwnKeyListSort>;

/**
 * The caller's own keys, in one table whatever their platform: search, a
 * filter per platform, newest last by default (the order they were added).
 * Built from the platforms, so the caller keeps it stable (useMemo).
 */
export function ownKeyListPreset(platforms: readonly KeyPlatform[]): OwnKeyListPreset {
  return {
    filters: [
      { value: "all", label: "myOrganization.apiKeys.allPlatforms", apply: [] },
      ...platforms.map((p) => ({ value: platformFilter(p.id), text: p.name })),
    ],
    sorts: [
      ...bothWays("createdAt", (dir) => `apiKeys.sort.createdAt-${dir}`, ["asc", "desc"]),
      ...bothWays("name", (dir) => `apiKeys.sort.name-${dir}`, ["asc", "desc"]),
      { by: "platform", dir: "asc", label: "apiKeys.sort.platform-asc" },
      { by: "platform", dir: "desc" },
      { by: "status", dir: "asc", label: "apiKeys.sort.status-asc" },
      { by: "status", dir: "desc" },
    ],
    defaultSort: { by: "createdAt", dir: "asc" },
  };
}
