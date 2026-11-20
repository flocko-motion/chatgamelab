import { config } from "@/config/env";

/**
 * Counts one page view of a share link for the admin statistics: at most once per browser,
 * link and day. Anonymous on the server (no account, no IP). Automated browsers are skipped;
 * link previews and most mail scanners never run this code at all.
 */
const KEY_PREFIX = "cgl_share_view_";

export function reportShareView(token: string): void {
  if (typeof navigator !== "undefined" && navigator.webdriver) return;
  const today = new Date().toISOString().slice(0, 10);
  try {
    if (localStorage.getItem(KEY_PREFIX + token) === today) return;
    localStorage.setItem(KEY_PREFIX + token, today);
  } catch {
    // storage unavailable: count anyway, a few double counts are acceptable
  }
  fetch(`${config.API_BASE_URL}/play/${token}/view`, { method: "POST", keepalive: true }).catch(() => {
    // statistics only
  });
}
