import { ObjRole, type ObjInstitution } from "@/api/generated";

/** Names of the organisation's heads, comma-separated. */
export function headNames(org: ObjInstitution): string {
  return (org.members ?? [])
    .filter((m) => m.role === ObjRole.RoleHead)
    .map((m) => m.name || m.email || "?")
    .join(", ");
}
