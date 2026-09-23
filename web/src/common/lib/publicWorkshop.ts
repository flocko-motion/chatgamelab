/**
 * The public workshop page lives at /w/<slug> (server: db/workshop_public.go).
 */

export const PUBLIC_WORKSHOP_PREFIX = "/w/";
export const PUBLIC_SLUG_MIN_LENGTH = 3;
export const PUBLIC_SLUG_MAX_LENGTH = 60;
export const PUBLIC_DESCRIPTION_MAX_LENGTH = 2000;
/** Sessions each game's play link on the page allows (server: db.PublicPageSessions). */
export const PUBLIC_PAGE_SESSIONS = 50;

const SLUG_PATTERN = /^[a-z0-9]+(-[a-z0-9]+)*$/;

export function publicWorkshopPath(slug: string): string {
  return `${PUBLIC_WORKSHOP_PREFIX}${slug}`;
}

/** Mirrors db.NormalizePublicSlug. */
export function normalizePublicSlug(input: string): string {
  return input.trim().toLowerCase();
}

/** Mirrors db.validatePublicSlug. */
export function isValidPublicSlug(slug: string): boolean {
  return (
    slug.length >= PUBLIC_SLUG_MIN_LENGTH &&
    slug.length <= PUBLIC_SLUG_MAX_LENGTH &&
    SLUG_PATTERN.test(slug)
  );
}
