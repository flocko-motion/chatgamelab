import { api } from "@/api/client/api";
import { ErrorCodes, extractRawErrorCode } from "@/common/types/errorCodes";

export type ParticipantLoginResult = "ok" | "locked" | "inactive" | "unknown" | "failed";

/**
 * Signs a workshop participant in with their token (whole, or its words);
 * the server sets the session cookie.
 */
export async function participantLogin(token: string): Promise<ParticipantLoginResult> {
  try {
    await api.postApiAuthParticipantLogin({ token });
    rememberParticipantCode(token);
    return "ok";
  } catch (error) {
    const code = extractRawErrorCode(error);
    if (code === ErrorCodes.TOKEN_LOCKED) return "locked";
    if (code === ErrorCodes.AUTH_WORKSHOP_INACTIVE) return "inactive";
    // An error with a code is the server's answer; none is a network failure.
    return code ? "unknown" : "failed";
  }
}

// The participant's code, kept in this browser only to show it again (on the
// inactive-workshop page) so the participant can sign in later.
const PARTICIPANT_CODE_KEY = "cgl_participant_token";

export function rememberParticipantCode(token: string): void {
  try {
    localStorage.setItem(PARTICIPANT_CODE_KEY, token);
  } catch {
    // Storage refused: nothing is kept.
  }
}

export function rememberedParticipantCode(): string | null {
  try {
    return localStorage.getItem(PARTICIPANT_CODE_KEY);
  } catch {
    return null;
  }
}

export function forgetParticipantCode(): void {
  try {
    localStorage.removeItem(PARTICIPANT_CODE_KEY);
  } catch {
    // Storage refused: nothing was kept.
  }
}
