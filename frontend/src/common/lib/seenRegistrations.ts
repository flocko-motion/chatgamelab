/**
 * The organisations registered lately this admin has been told about, in
 * this browser, so the notice of new organisations opens once per
 * organisation.
 */
const SEEN_KEY = "cgl_seen_institution_applications";

/** The ids of the organisations this admin has been told about. */
export function seenRegistrationIds(): Set<string> {
  try {
    const stored = localStorage.getItem(SEEN_KEY);
    return new Set(stored ? (JSON.parse(stored) as string[]) : []);
  } catch {
    return new Set();
  }
}

/** Notes that this admin has been told about the organisations. */
export function markRegistrationsSeen(ids: string[]): void {
  try {
    const seen = seenRegistrationIds();
    ids.forEach((id) => seen.add(id));
    localStorage.setItem(SEEN_KEY, JSON.stringify([...seen]));
  } catch {
    // storage unavailable: the notice opens again next time
  }
}
