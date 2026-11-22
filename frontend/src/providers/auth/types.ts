/**
 * Shared types for the auth module.
 */
import type { UserView } from "@/api/types";
import type { CallerStanding, Registration } from "./useCaller";

export interface AuthUser {
  sub: string;
  name?: string;
  email?: string;
  role?: string;
}

/** Suggestions that pre-fill the registration form, from the sign-in. */
export interface RegistrationData {
  name: string;
  email: string;
}

export interface AuthContextType {
  /** The signed-in account, in brief; null while anonymous or registering. */
  user: AuthUser | null;
  /** The signed-in account (from /api/users/me). */
  backendUser: UserView | null;
  isLoading: boolean;
  /** True once a registered account or a Guest is signed in. */
  isAuthenticated: boolean;
  /** True if the account is a Guest. */
  isGuest: boolean;
  /** True while a signed-in account has not completed registration. */
  needsRegistration: boolean;
  /** Suggestions to pre-fill the registration form. */
  registrationData: RegistrationData | null;
  /** Error fetching the account: the app is not operational. */
  backendError: string | null;
  /** True if the workshop of a Guest in a workshop is inactive. */
  isWorkshopInactive: boolean;
  /** True if this instance offers Auth0 sign-in. */
  auth0Enabled: boolean;
  loginWithAuth0: (options?: { screenHint?: string }) => void;
  /** Development only: true where the preset dev accounts can sign in. */
  isDevMode: boolean;
  /** Development only: signs in as a seeded account by preset role ("admin-1", …). */
  loginWithRole: (role: string) => Promise<void>;
  logout: () => Promise<void> | void;
  /** Fetch the account again, after an error or a change. */
  retryBackendFetch: () => void;
  /** Complete the registration of the signed-in account. */
  register: (registration: Registration) => Promise<void>;
  /** The caller's standing with organisations, from /api/users/me. */
  standing: CallerStanding;
  /** Fetch the account again and wait for it, after a change. */
  refreshCaller: () => Promise<void>;
}
