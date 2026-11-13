/**
 * Remembers across the Auth0 redirect what a person chose on the registration page
 * ("Persönliches Konto" or "Als Organisation registrieren"). Without a stored choice the
 * app asks after the first login.
 */
const KEY = "cgl_registration_intent";

export type RegistrationIntent = "organization" | "individual";

export function setRegistrationIntent(intent: RegistrationIntent): void {
  try {
    localStorage.setItem(KEY, intent);
  } catch {
    // storage unavailable: the app asks again after the login
  }
}

export function getRegistrationIntent(): RegistrationIntent | null {
  try {
    const v = localStorage.getItem(KEY);
    return v === "organization" || v === "individual" ? v : null;
  } catch {
    return null;
  }
}

export function clearRegistrationIntent(): void {
  try {
    localStorage.removeItem(KEY);
  } catch {
    // ignore
  }
}
