import type { WorkshopListFilter, WorkshopListSort } from "@/api/hooks";
import type { ListPreset } from "@/api/types";

/**
 * The organisation's workshop list: a toggle that hides the inactive ones,
 * its orders (newest first by default) and its columns.
 */
export type WorkshopColumnKey = "name" | "status" | "participants" | "created" | "actions";

export const WORKSHOP_LIST_PRESET: ListPreset<string, WorkshopListSort, WorkshopListFilter> = {
  toggles: [{ value: "active", label: "myOrganization.workshops.hideInactive" }],
  sorts: [
    { by: "createdAt", dir: "desc", label: "myOrganization.workshops.sort.createdAtDesc" },
    { by: "createdAt", dir: "asc", label: "myOrganization.workshops.sort.createdAtAsc" },
    { by: "name", dir: "asc", label: "myOrganization.workshops.sort.nameAsc" },
    { by: "name", dir: "desc", label: "myOrganization.workshops.sort.nameDesc" },
    {
      by: "participantCount",
      dir: "desc",
      label: "myOrganization.workshops.sort.participantsDesc",
    },
    { by: "participantCount", dir: "asc", label: "myOrganization.workshops.sort.participantsAsc" },
  ],
  defaultSort: { by: "createdAt", dir: "desc" },
};

export const WORKSHOP_COLUMNS: WorkshopColumnKey[] = [
  "name",
  "status",
  "participants",
  "created",
  "actions",
];
