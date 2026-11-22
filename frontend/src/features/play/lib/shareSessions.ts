/**
 * The session a share-link guest last played through a link, in this
 * browser. A guest has no list of their sessions; its id is their only
 * handle on it, so the web app keeps it per token.
 */
export const shareSessions = {
  get(token: string): string | null {
    try {
      return localStorage.getItem(PREFIX + token);
    } catch {
      return null;
    }
  },
  remember(token: string, id: string): void {
    try {
      localStorage.setItem(PREFIX + token, id);
    } catch {
      // Kept nowhere: the next visit through the link starts anew.
    }
  },
  forget(token: string): void {
    try {
      localStorage.removeItem(PREFIX + token);
    } catch {
      // Nothing kept, nothing to forget.
    }
  },
};

const PREFIX = "cgl.share.";
