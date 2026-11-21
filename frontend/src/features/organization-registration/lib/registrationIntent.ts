/**
 * What a person chose on the registration page, an individual account or an
 * organisation, kept across the Auth0 redirect. Without a choice kept, the
 * app asks after the first sign-in.
 */
const KEY = "cgl_registration_intent";

export type RegistrationIntent = "organization" | "individual";

/** Keeps the choice for the registration after the sign-in. */
export function setRegistrationIntent(intent: RegistrationIntent): void {
  try {
    localStorage.setItem(KEY, intent);
  } catch {
    // storage unavailable: the app asks again after the sign-in
  }
}

/** The choice kept, if any. */
export function getRegistrationIntent(): RegistrationIntent | null {
  try {
    const intent = localStorage.getItem(KEY);
    return intent === "organization" || intent === "individual" ? intent : null;
  } catch {
    return null;
  }
}

/** Forgets the choice, so a later new account in this browser asks again. */
export function clearRegistrationIntent(): void {
  try {
    localStorage.removeItem(KEY);
  } catch {
    // storage unavailable: nothing was kept
  }
}
