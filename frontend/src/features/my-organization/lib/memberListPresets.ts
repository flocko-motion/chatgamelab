import type { MemberListSort } from "@/api/hooks";
import { bothWays } from "@/api/lists";
import type { ListPreset } from "@/api/types";

/**
 * The member lists of the organisation page. Whoever sees the members' roles
 * (everyone with a role in the organisation) gets the full list; someone
 * without one sees the names only.
 */
export type MemberListPresetName = "full" | "names";

export type MemberColumnKey = "name" | "email" | "role" | "actions";

export interface MemberListPresetConfig {
  list: ListPreset<string, MemberListSort>;
  columns: MemberColumnKey[];
}

const byName = bothWays("name", (dir) => `myOrganization.sort.name-${dir}`, ["asc", "desc"]);

export const MEMBER_LIST_PRESETS: Record<MemberListPresetName, MemberListPresetConfig> = {
  full: {
    list: {
      sorts: [
        ...byName,
        ...bothWays("role", (dir) => `myOrganization.sort.role-${dir}`, ["asc", "desc"]),
        // The email header sorts; the sort menu doesn't offer it.
        ...bothWays("email", undefined, ["asc", "desc"]),
      ],
      defaultSort: { by: "name", dir: "asc" },
    },
    columns: ["name", "email", "role", "actions"],
  },
  names: {
    list: { sorts: byName, defaultSort: { by: "name", dir: "asc" } },
    columns: ["name", "actions"],
  },
};
