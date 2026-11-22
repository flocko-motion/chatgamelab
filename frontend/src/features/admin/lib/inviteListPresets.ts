import type { InviteListSort } from "@/api/hooks";
import { bothWays } from "@/api/lists";
import type { ListPreset } from "@/api/types";

/**
 * The invite lists: every invite (admins, with each invite's organisation)
 * or one organisation's. Both share a page with another list, so their query
 * stays out of the URL.
 */
export type InviteListPreset = "all" | "organisation";

export type InviteColumnKey = "organisation" | "role" | "invited" | "status" | "modified" | "actions";

export interface InviteListPresetConfig {
  list: ListPreset<string, InviteListSort>;
  columns: InviteColumnKey[];
}

const sortLabel = (key: string) => (dir: string) => `admin.invites.sort.${key}-${dir}`;

const BASE_SORTS = [
  ...bothWays<InviteListSort>("modifiedAt", sortLabel("modifiedAt")),
  ...bothWays<InviteListSort>("status", sortLabel("status"), ["asc", "desc"]),
  ...bothWays<InviteListSort>("role", sortLabel("role"), ["asc", "desc"]),
];

export const INVITE_LIST_PRESETS: Record<InviteListPreset, InviteListPresetConfig> = {
  all: {
    list: {
      sorts: [
        ...BASE_SORTS,
        ...bothWays<InviteListSort>("organisation", sortLabel("organization"), ["asc", "desc"]),
      ],
      defaultSort: { by: "modifiedAt", dir: "desc" },
      inUrl: false,
    },
    columns: ["organisation", "role", "invited", "status", "modified", "actions"],
  },
  organisation: {
    list: {
      sorts: BASE_SORTS,
      defaultSort: { by: "modifiedAt", dir: "desc" },
      inUrl: false,
    },
    columns: ["role", "invited", "status", "modified", "actions"],
  },
};
