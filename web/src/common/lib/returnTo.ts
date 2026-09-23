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
