import type { OrganisationView } from "@/api/generated/api.gen";
import { RoleName } from "@/common/lib/roles";

/** The names of the organisation's heads, comma-separated. */
export function headNames(org: OrganisationView): string {
  return (org.members ?? [])
    .filter((m) => m.role?.role === RoleName.Head)
    .map((m) => m.name || m.email || "?")
    .join(", ");
}
