/**
 * The page to open after the next login or registration, kept for this tab.
 * sessionStorage survives the round trip through Auth0.
 */

const RETURN_TO_KEY = "cgl_return_to";

function isLocalPath(path: string | null): path is string {
  return !!path && path.startsWith("/") && !path.startsWith("//");
}

export function rememberReturnTo(path: string) {
  if (!isLocalPath(path)) return;
  try {
    sessionStorage.setItem(RETURN_TO_KEY, path);
  } catch {
    // Storage blocked: the visitor lands on the dashboard instead.
  }
}

/** Reads the remembered page and keeps it for a later step. */
export function peekReturnTo(): string | null {
  try {
    const path = sessionStorage.getItem(RETURN_TO_KEY);
    return isLocalPath(path) ? path : null;
  } catch {
    return null;
  }
}

/** Reads the remembered page and forgets it. */
export function takeReturnTo(): string | null {
  const path = peekReturnTo();
  try {
    sessionStorage.removeItem(RETURN_TO_KEY);
  } catch {
    // Nothing to clean up.
  }
  return path;
}

const COPY_INTENT_KEY = "cgl_public_copy";

interface CopyIntent {
  slug: string;
  gameId: string;
}

/** The game a visitor of /w/<slug> wanted to copy before logging in. */
export function rememberCopyIntent(intent: CopyIntent) {
  try {
    sessionStorage.setItem(COPY_INTENT_KEY, JSON.stringify(intent));
  } catch {
    // Storage blocked: the visitor clicks "Kopieren" again.
  }
}

/** Reads the game to copy on this page and forgets any remembered one. */
export function takeCopyIntent(slug: string): string | null {
  try {
    const raw = sessionStorage.getItem(COPY_INTENT_KEY);
    sessionStorage.removeItem(COPY_INTENT_KEY);
    const intent = raw ? (JSON.parse(raw) as Partial<CopyIntent>) : null;
    return intent?.slug === slug && typeof intent.gameId === "string"
      ? intent.gameId
      : null;
  } catch {
    return null;
  }
}
