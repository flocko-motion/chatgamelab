/**
 * Shared types for the auth module.
 */
import type { ObjUser } from "@/api/generated";

export interface AuthUser {
  sub: string;
  name?: string;
  email?: string;
  picture?: string;
  role?: string;
}

/** Data needed for registration when user is authenticated but not registered */
export interface RegistrationData {
  auth0Id: string;
  email: string;
  name: string;
}

export interface AuthContextType {
  /** Auth0 user info (from token) */
  user: AuthUser | null;
  /** Backend user data (from /api/users/me) */
  backendUser: ObjUser | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  /** True if user is authenticated via participant cookie (workshop participant) */
  isParticipant: boolean;
  /** True if user is authenticated with Auth0 but not registered in backend */
  needsRegistration: boolean;
  /** Data from Auth0 to pre-fill registration form */
  registrationData: RegistrationData | null;
  /** Error fetching backend user - app is not operational */
  backendError: string | null;
  /** True if participant's workshop is inactive */
  isWorkshopInactive: boolean;
  loginWithAuth0: () => void;
  loginWithRole: (role: string) => void;
  logout: () => Promise<void> | void;
  isDevMode: boolean;
  /** Get the current access token for API calls. Returns null if not authenticated (participants use cookies). */
  getAccessToken: () => Promise<string | null>;
  /** Retry fetching backend user after an error */
  retryBackendFetch: () => void;
  /** Register the user with the backend */
  register: (name: string, email: string) => Promise<void>;
}
