import type { OrganisationListFilter, OrganisationListSort } from "@/api/hooks";
import { bothWays } from "@/api/lists";
import type { ListPreset } from "@/api/types";

/**
 * The organisation lists of the admins' page: every organisation (the page's
 * main list, its query in the URL), and those that registered themselves
 * lately (embedded, refreshing itself).
 */
export type OrganisationColumnKey = "name" | "heads" | "status" | "members" | "actions";

export type RecentRegistrationColumnKey = "registered" | "organisation" | "person" | "status";

export const ORGANISATION_LIST_PRESET: {
  list: ListPreset<string, OrganisationListSort>;
  columns: OrganisationColumnKey[];
} = {
  list: {
    sorts: [
      ...bothWays<OrganisationListSort>("name", (dir) => `games.sort.name-${dir}`, ["asc", "desc"]),
      ...bothWays<OrganisationListSort>("createdAt", (dir) => `games.sort.createdAt-${dir}`),
    ],
    defaultSort: { by: "name", dir: "asc" },
  },
  columns: ["name", "heads", "status", "members", "actions"],
};

/** How often the recent registrations refresh: new ones arrive while the page is open. */
const RECENT_REFRESH_MS = 2 * 60 * 1000;

export const RECENT_REGISTRATION_PRESET: {
  list: ListPreset<string, OrganisationListSort>;
  columns: RecentRegistrationColumnKey[];
} = {
  list: {
    base: ["recentlyRegistered" satisfies OrganisationListFilter],
    // header-only orders: the embedded list has no sort menu
    sorts: [...bothWays<OrganisationListSort>("registeredAt"), ...bothWays<OrganisationListSort>("name")],
    defaultSort: { by: "registeredAt", dir: "desc" },
    inUrl: false,
    refetchInterval: RECENT_REFRESH_MS,
  },
  columns: ["registered", "organisation", "person", "status"],
};
