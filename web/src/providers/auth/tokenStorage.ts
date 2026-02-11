/**
 * Token storage utilities for participant and dev mode authentication.
 * Pure functions with no React dependencies.
 */

// ── Participant Token ─────────────────────────────────────────────────

const PARTICIPANT_TOKEN_STORAGE_KEY = "cgl_participant_token";

export function getStoredParticipantToken(): string | null {
  try {
    return localStorage.getItem(PARTICIPANT_TOKEN_STORAGE_KEY);
  } catch {
    return null;
  }
}

export function storeParticipantToken(token: string): void {
  try {
    localStorage.setItem(PARTICIPANT_TOKEN_STORAGE_KEY, token);
  } catch {
    // Ignore storage errors
  }
}

export function clearParticipantToken(): void {
  try {
    localStorage.removeItem(PARTICIPANT_TOKEN_STORAGE_KEY);
  } catch {
    // Ignore storage errors
  }
}

// ── Dev Mode Token ────────────────────────────────────────────────────

const DEV_TOKEN_STORAGE_KEY = "cgl_dev_token";

export interface TokenCache {
  token: string;
  expiresAt: number;
  userId: string;
  role: string;
}

export function getStoredDevToken(): TokenCache | null {
  try {
    const stored = localStorage.getItem(DEV_TOKEN_STORAGE_KEY);
    if (!stored) return null;
    const parsed = JSON.parse(stored) as TokenCache;
    // Check if token is still valid
    if (parsed.expiresAt > Date.now()) {
      return parsed;
    }
    // Token expired, clean up
    localStorage.removeItem(DEV_TOKEN_STORAGE_KEY);
    return null;
  } catch {
    return null;
  }
}

export function storeDevToken(cache: TokenCache): void {
  try {
    localStorage.setItem(DEV_TOKEN_STORAGE_KEY, JSON.stringify(cache));
  } catch {
    // Ignore storage errors
  }
}

export function clearStoredDevToken(): void {
  try {
    localStorage.removeItem(DEV_TOKEN_STORAGE_KEY);
  } catch {
    // Ignore storage errors
  }
}
