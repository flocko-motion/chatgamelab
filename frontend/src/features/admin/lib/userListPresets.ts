import type { UserListFilter, UserListSort } from "@/api/hooks";
import { bothWays } from "@/api/lists";
import type { ListPreset } from "@/api/types";

/**
 * The admins' user list: what it offers and its columns. Who may do what
 * with a user is not the preset's business (useUserManagement).
 */
export type UserColumnKey = "name" | "email" | "role" | "organisation" | "joined" | "actions";

const sortLabel = (key: string) => (dir: string) => `admin.users.sort.${key}-${dir}`;

export const USER_LIST_PRESET: {
  list: ListPreset<string, UserListSort>;
  columns: UserColumnKey[];
} = {
  list: {
    // The first is the default: guests stay out unless asked for.
    filters: [
      { value: "noGuests" satisfies UserListFilter, label: "admin.users.filters.noGuests" },
      { value: "all", label: "admin.users.filters.all", apply: [] },
    ],
    sorts: [
      ...bothWays("name", sortLabel("name"), ["asc", "desc"]),
      ...bothWays("email", sortLabel("email"), ["asc", "desc"]),
      ...bothWays("role", sortLabel("role"), ["asc", "desc"]),
      ...bothWays("organisation", sortLabel("organization"), ["asc", "desc"]),
      ...bothWays("createdAt", sortLabel("joined")),
    ],
    defaultSort: { by: "name", dir: "asc" },
  },
  columns: ["name", "email", "role", "organisation", "joined", "actions"],
};
