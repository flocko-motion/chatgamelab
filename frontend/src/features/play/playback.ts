/** What a play route opens: a new session for a game or a share link, or a session. */
export type PlaybackTarget =
  /** advanced: the starter adapts the session's prompts before it starts. */
  | { readonly kind: "game"; readonly id: string; readonly advanced?: boolean }
  | { readonly kind: "share"; readonly id: string }
  | { readonly kind: "session"; readonly id: string };

/** The play routes, each with what it opens; null for every other page. */
export function playbackTarget(pathname: string): PlaybackTarget | null {
  const game = pathname.match(/^\/games\/([^/]+)\/play(\/advanced)?\/?$/);
  if (game) return { kind: "game", id: decodeURIComponent(game[1]), advanced: !!game[2] };
  const share = pathname.match(/^\/play\/([^/]+)\/?$/);
  if (share) return { kind: "share", id: decodeURIComponent(share[1]) };
  const session = pathname.match(/^\/sessions\/([^/]+)\/?$/);
  if (session) return { kind: "session", id: decodeURIComponent(session[1]) };
  return null;
}

/** Where the engine is mounted: in-process under the API, on the page's origin. */
export function engineRoot(): URL {
  return new URL("/api/engine/", window.location.origin);
}

/** The playback-engine's address for a session the web app has opened, in the player's language. */
export function playbackURL(sessionId: string, language: string): string {
  const url = new URL("playback-engine/", engineRoot());
  url.searchParams.set("session", sessionId);
  url.searchParams.set("lang", language);
  return url.toString();
}
