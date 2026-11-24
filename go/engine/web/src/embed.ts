/**
 * The playback-engine embedded on another website (#340): the host's own
 * CSS, handed over by the loader script or named in the address. Headless;
 * ui/embed.ts applies it to the page. The styling contract is
 * docs/embed-styling.md.
 */

/** What the session's own address carries over from the page's, so a reload looks the same. */
const KEPT = ["css", "lang"] as const;

/** The address a started session replaces the page's with: its ID, and how the page looks. */
export function sessionAddress(id: string, params: URLSearchParams): string {
  const next = new URLSearchParams({ session: id });
  for (const name of KEPT) {
    const value = params.get(name);
    if (value) next.set(name, value);
  }
  return `?${next.toString()}`;
}

/** The most CSS a host may hand over, in characters. */
const MAX_CSS = 100_000;

/** The CSS in a message from the host page, or null when it is none. */
export function handedCSS(data: unknown): string | null {
  if (typeof data !== "object" || data === null) return null;
  const message = data as { type?: unknown; css?: unknown };
  if (message.type !== "cgl-css" || typeof message.css !== "string") return null;
  return message.css.length <= MAX_CSS ? message.css : null;
}

/** The host's stylesheet the address names, if it is an https address. */
export function stylesheetURL(params: URLSearchParams): string | null {
  const raw = params.get("css");
  if (!raw) return null;
  try {
    const url = new URL(raw);
    return url.protocol === "https:" ? url.toString() : null;
  } catch {
    return null;
  }
}
