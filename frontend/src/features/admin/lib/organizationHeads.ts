import type { OrganisationView } from "@/api/types";
import { RoleName } from "@/common/lib/roles";

/** The names of the organisation's heads, comma-separated. */
export function headNames(org: OrganisationView): string {
  return (org.members ?? [])
    .filter((m) => m.role?.role === RoleName.Head)
    .map((m) => m.name || m.email || "?")
    .join(", ");
}

/** The organisation's operator and city, as one line ("Operator · City"); empty if neither. */
export function operatorAndCity(org: Pick<OrganisationView, "operator" | "city">): string {
  return [org.operator, org.city].filter(Boolean).join(" · ");
}
