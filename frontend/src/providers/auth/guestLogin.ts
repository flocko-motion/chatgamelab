import { signInGuest } from "@/api/hooks/authentication";
import { ErrorCodes, extractRawErrorCode } from "@/common/types/errorCodes";

export type GuestLoginResult = "ok" | "locked" | "inactive" | "unknown" | "failed";

/**
 * Signs a Guest in a workshop in with their token (whole, or its words);
 * the server sets the session cookie.
 */
export async function guestLogin(token: string): Promise<GuestLoginResult> {
  try {
    await signInGuest(token);
    rememberGuestCode(token);
    return "ok";
  } catch (error) {
    const code = extractRawErrorCode(error);
    if (code === ErrorCodes.TOKEN_LOCKED) return "locked";
    if (code === ErrorCodes.AUTH_WORKSHOP_INACTIVE) return "inactive";
    // An error with a code is the server's answer; none is a network failure.
    return code ? "unknown" : "failed";
  }
}

// The code of a Guest in a workshop, kept in this browser only to show it again (on the
// inactive-workshop page) so the Guest can sign in later.
const GUEST_CODE_KEY = "cgl_guest_token";
// The key browsers kept the code under before the role was called Guest; a code
// found there moves to GUEST_CODE_KEY.
const OLD_GUEST_CODE_KEY = "cgl_participant_token";

export function rememberGuestCode(token: string): void {
  try {
    localStorage.setItem(GUEST_CODE_KEY, token);
    localStorage.removeItem(OLD_GUEST_CODE_KEY);
  } catch {
    // Storage refused: nothing is kept.
  }
}

export function rememberedGuestCode(): string | null {
  try {
    const code = localStorage.getItem(GUEST_CODE_KEY);
    if (code) return code;
    const old = localStorage.getItem(OLD_GUEST_CODE_KEY);
    if (old) rememberGuestCode(old);
    return old;
  } catch {
    return null;
  }
}

export function forgetGuestCode(): void {
  try {
    localStorage.removeItem(GUEST_CODE_KEY);
    localStorage.removeItem(OLD_GUEST_CODE_KEY);
  } catch {
    // Storage refused: nothing was kept.
  }
}
