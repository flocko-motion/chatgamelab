/** What a play route opens: a new session for a game or a share link, or a session. */
export type PlaybackTarget =
  | { readonly kind: "game"; readonly id: string }
  | { readonly kind: "share"; readonly id: string }
  | { readonly kind: "session"; readonly id: string };

/** The play routes, each with what it opens; null for every other page. */
export function playbackTarget(pathname: string): PlaybackTarget | null {
  const game = pathname.match(/^\/games\/([^/]+)\/play\/?$/);
  if (game) return { kind: "game", id: decodeURIComponent(game[1]) };
  const share = pathname.match(/^\/play\/([^/]+)\/?$/);
  if (share) return { kind: "share", id: decodeURIComponent(share[1]) };
  const session = pathname.match(/^\/sessions\/([^/]+)\/?$/);
  if (session) return { kind: "session", id: decodeURIComponent(session[1]) };
  return null;
}

/**
 * Where the engine is mounted: in-process under the API, on the page's origin,
 * which the playback-engine needs to tell its parent page about a new session.
 */
export function engineRoot(): URL {
  return new URL("/api/engine/", window.location.origin);
}

/** The playback-engine's address for a target, in the player's language. */
export function playbackURL(target: PlaybackTarget, language: string): string {
  const url = new URL("playback-engine/", engineRoot());
  if (target.kind === "session") {
    url.searchParams.set("session", target.id);
  } else {
    url.searchParams.set("new", "");
    url.searchParams.set(target.kind, target.id);
  }
  url.searchParams.set("lang", language);
  return url.toString();
}

/**
 * Forgets the session the playback-engine remembers for a game in this
 * browser, so starting the game anew does not offer to continue it. The
 * playback-engine shares the page's origin and keeps it in localStorage under
 * this key (engine/web/src/launch.ts).
 */
export function forgetRememberedSession(gameId: string): void {
  try {
    localStorage.removeItem(`cgl.session.game:${gameId}`);
  } catch {
    // Storage refused: nothing was kept.
  }
}
