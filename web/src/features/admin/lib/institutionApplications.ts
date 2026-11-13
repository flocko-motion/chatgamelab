import type { ObjInstitutionApplication } from "@/api/generated";

/** React Query key of the admin list of organisation applications. */
export const INSTITUTION_APPLICATIONS_QUERY_KEY = [
  "institutionApplications",
] as const;

/** How far back the admin bell looks for newly registered organisations. */
export const NEW_ORGANISATION_DAYS = 14;

/**
 * A registration that created an organisation within the last NEW_ORGANISATION_DAYS days.
 * Stage 1 has no e-mail confirmation, so every registration is confirmed right away;
 * admins check new organisations and can lock them if needed.
 */
export function isNewOrganisation(a: ObjInstitutionApplication, now = Date.now()): boolean {
  if (a.status !== "confirmed" || !a.confirmedAt) return false;
  return now - new Date(a.confirmedAt).getTime() < NEW_ORGANISATION_DAYS * 24 * 60 * 60 * 1000;
}

const SEEN_KEY = "cgl_seen_institution_applications";

/** Application IDs this admin has already been told about (per browser). */
export function getSeenApplicationIds(): Set<string> {
  try {
    const stored = localStorage.getItem(SEEN_KEY);
    return new Set(stored ? (JSON.parse(stored) as string[]) : []);
  } catch {
    return new Set();
  }
}

export function markApplicationsSeen(ids: string[]): void {
  try {
    const seen = getSeenApplicationIds();
    ids.forEach((id) => seen.add(id));
    localStorage.setItem(SEEN_KEY, JSON.stringify([...seen]));
  } catch {
    // Ignore storage errors
  }
}
