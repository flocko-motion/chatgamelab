import type { VerifiedPageListFilter, VerifiedPageListSort } from "@/api/hooks";
import { bothWays } from "@/api/lists";
import type { ListPreset } from "@/api/types";

/** The admins' allow list of verified pages: waiting first, filterable by approval. */
export type VerifiedPageColumnKey = "organisation" | "page" | "status" | "actions";

export const VERIFIED_PAGE_LIST_PRESET: {
  list: ListPreset<VerifiedPageListFilter | "all", VerifiedPageListSort>;
  columns: VerifiedPageColumnKey[];
} = {
  list: {
    filters: [
      { value: "all", label: "admin.verifiedPages.filters.all", apply: [] },
      { value: "waiting", label: "admin.verifiedPages.filters.waiting" },
      { value: "approved", label: "admin.verifiedPages.filters.approved" },
    ],
    sorts: [...bothWays<VerifiedPageListSort>("status"), ...bothWays<VerifiedPageListSort>("organisation")],
    defaultSort: { by: "status", dir: "asc" },
    inUrl: false,
  },
  columns: ["organisation", "page", "status", "actions"],
};
