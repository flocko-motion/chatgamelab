import type { ObjInstitutionApplication } from "@/api/generated";

/** React Query key of the admin list of organisation applications. */
export const INSTITUTION_APPLICATIONS_QUERY_KEY = [
  "institutionApplications",
] as const;

/** An application still waiting for its confirmation (not confirmed, not anonymised, not lapsed). */
export function isOpenApplication(a: ObjInstitutionApplication): boolean {
  return a.status === "pending" && !a.anonymizedAt;
}

/** An application an admin can create a fresh confirmation link for. */
export function canCreateLink(a: ObjInstitutionApplication): boolean {
  return (
    (a.status === "pending" || a.status === "expired") &&
    !a.anonymizedAt &&
    !!a.email
  );
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
