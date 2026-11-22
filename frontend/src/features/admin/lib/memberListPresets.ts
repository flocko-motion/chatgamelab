import type { MemberListSort } from "@/api/hooks";
import { bothWays } from "@/api/lists";
import type { ListPreset } from "@/api/types";

/**
 * An organisation's members as the admins' organisation page shows them,
 * embedded in the organisation's row: no toolbar, its orders in the headers.
 */
export type MemberColumnKey = "name" | "email" | "role" | "actions";

export const MEMBER_LIST_PRESET: {
  list: ListPreset<string, MemberListSort>;
  columns: MemberColumnKey[];
} = {
  list: {
    sorts: [...bothWays<MemberListSort>("name"), ...bothWays<MemberListSort>("email"), ...bothWays<MemberListSort>("role")],
    defaultSort: { by: "name", dir: "asc" },
    inUrl: false,
  },
  columns: ["name", "email", "role", "actions"],
};
