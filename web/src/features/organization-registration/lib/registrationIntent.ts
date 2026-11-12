/**
 * Remembers across the Auth0 redirect that a person chose "register as organisation".
 */
const KEY = "cgl_registration_intent";

export type RegistrationIntent = "organization";

export function setRegistrationIntent(intent: RegistrationIntent): void {
  try {
    localStorage.setItem(KEY, intent);
  } catch {
    // storage unavailable: the person lands in the normal registration form
  }
}

export function getRegistrationIntent(): RegistrationIntent | null {
  try {
    return localStorage.getItem(KEY) === "organization" ? "organization" : null;
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
