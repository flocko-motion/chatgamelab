import type { WorkshopListSort } from "@/api/hooks";
import type { ListPreset } from "@/api/types";

/** The five active workshops of the caller's organisation changed last. */
export const RECENT_WORKSHOPS_PRESET: ListPreset<string, WorkshopListSort> = {
  base: ["active"],
  sorts: [{ by: "modifiedAt", dir: "desc" }],
  defaultSort: { by: "modifiedAt", dir: "desc" },
  inUrl: false,
  defaultPageSize: 5,
};
