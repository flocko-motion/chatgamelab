/* eslint-disable react-refresh/only-export-components */
import React, {
  createContext,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { useAuth0 } from "@auth0/auth0-react";
import { useTranslation } from "react-i18next";
import { auth0Config } from "../config/auth0";
import { config } from "../config/env";
import { Api } from "../api/generated";
import { createAuthenticatedApiConfig, getApiConfig } from "../api/client/http";
import { authLogger } from "../config/logger";
import { ROUTES } from "../common/routes/routes";
import {
  ErrorCodes,
  extractErrorCode,
  extractRawErrorCode,
} from "../common/types/errorCodes";
import type { ObjUser } from "../api/generated";

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
  logout: () => void;
  isDevMode: boolean;
  /** Get the current access token for API calls. Returns null if not authenticated (participants use cookies). */
  getAccessToken: () => Promise<string | null>;
  /** Retry fetching backend user after an error */
  retryBackendFetch: () => void;
  /** Register the user with the backend */
  register: (name: string, email: string) => Promise<void>;
}

export const AuthContext = createContext<AuthContextType | undefined>(
  undefined,
);

// Default context value for error boundary recovery scenarios
const defaultAuthContext: AuthContextType = {
  user: null,
  backendUser: null,
  isLoading: true,
  isAuthenticated: false,
  isParticipant: false,
  needsRegistration: false,
  registrationData: null,
  backendError: null,
  isWorkshopInactive: false,
  loginWithAuth0: () => {},
  loginWithRole: () => {},
  logout: () => {},
  isDevMode: false,
  getAccessToken: async () => null,
  retryBackendFetch: () => {},
  register: async () => {},
};

export function useAuth() {
  const context = React.useContext(AuthContext);
  // Return default context during error boundary recovery to prevent cascading errors
  // This can happen when TanStack Router's CatchBoundaryImpl tries to re-render
  if (!context) {
    authLogger.warning(
      "useAuth called outside AuthProvider context, returning default (error boundary recovery)",
    );
    return defaultAuthContext;
  }
  return context;
}

interface AuthProviderProps {
  children: React.ReactNode;
}

// Token cache for dev mode
interface TokenCache {
  token: string;
  expiresAt: number;
  userId: string;
  role: string;
}

const DEV_TOKEN_STORAGE_KEY = "cgl_dev_token";

// Helper to get dev token from localStorage
function getStoredDevToken(): TokenCache | null {
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

// Helper to store dev token in localStorage
function storeDevToken(cache: TokenCache): void {
  try {
    localStorage.setItem(DEV_TOKEN_STORAGE_KEY, JSON.stringify(cache));
  } catch {
    // Ignore storage errors
  }
}

// Helper to clear dev token from localStorage
function clearStoredDevToken(): void {
  try {
    localStorage.removeItem(DEV_TOKEN_STORAGE_KEY);
  } catch {
    // Ignore storage errors
  }
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const { t } = useTranslation();
  const {
    user: auth0User,
    isAuthenticated: auth0IsAuthenticated,
    isLoading: auth0IsLoading,
    loginWithRedirect: auth0LoginWithRedirect,
    logout: auth0Logout,
    getAccessTokenSilently,
  } = useAuth0();

  const [user, setUser] = useState<AuthUser | null>(null);
  const [backendUser, setBackendUser] = useState<ObjUser | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isParticipant, setIsParticipant] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [backendError, setBackendError] = useState<string | null>(null);
  const [needsRegistration, setNeedsRegistration] = useState(false);
  const [registrationData, setRegistrationData] =
    useState<RegistrationData | null>(null);
  const [isWorkshopInactive, setIsWorkshopInactive] = useState(false);
  const [isDevMode] = useState(
    import.meta.env.VITE_DEV_MODE === "true" || import.meta.env.DEV,
  );

  // Token cache for dev mode tokens (initialize from localStorage)
  const devTokenCache = useRef<TokenCache | null>(getStoredDevToken());

  // Track if we've already fetched the backend user to avoid duplicate calls
  const backendFetchAttempted = useRef(false);

  // Get access token function (defined early so it can be used in effects)
  const getAccessToken = useCallback(async (): Promise<string | null> => {
    // Participants use cookie auth, not token auth
    if (isParticipant) {
      return null;
    }

    // Auth0 authenticated - use Auth0 SDK (handles caching internally)
    if (auth0IsAuthenticated) {
      try {
        const token = await getAccessTokenSilently({
          authorizationParams: {
            audience: auth0Config.audience,
          },
        });
        return token;
      } catch (error) {
        authLogger.error("Failed to get Auth0 access token", { error });
        return null;
      }
    }

    // Dev mode - return cached token if valid
    if (devTokenCache.current && devTokenCache.current.expiresAt > Date.now()) {
      return devTokenCache.current.token;
    }

    // No valid token available
    return null;
  }, [auth0IsAuthenticated, getAccessTokenSilently, isParticipant]);

  // Check if error is a "user not registered" response
  const isUserNotRegisteredError = (error: unknown): boolean => {
    return extractErrorCode(error) === ErrorCodes.USER_NOT_REGISTERED;
  };

  // Get registration data from Auth0 user - apply smart defaults for name
  const getRegistrationDataFromAuth0 =
    useCallback((): RegistrationData | null => {
      if (!auth0User?.sub) return null;

      const email = auth0User.email || "";
      let name = "";

      // Use nickname or name from Auth0, but skip if it looks like an email
      const isEmailLike = (s: string) =>
        s.includes("@") || s.includes("+") || s === email.split("@")[0];

      if (auth0User.nickname && !isEmailLike(auth0User.nickname)) {
        name = auth0User.nickname;
      } else if (auth0User.name && !isEmailLike(auth0User.name)) {
        name = auth0User.name;
      }

      return {
        auth0Id: auth0User.sub,
        email,
        name,
      };
    }, [auth0User]);

  // Fetch backend user
  const fetchBackendUser = useCallback(async () => {
    try {
      setBackendError(null);
      setNeedsRegistration(false);
      setRegistrationData(null);
      const api = new Api(createAuthenticatedApiConfig(getAccessToken));
      const response = await api.users.getUsers();
      setBackendUser(response.data);
      authLogger.debug("Backend user fetched", {
        userId: response.data.id,
        name: response.data.name,
      });
    } catch (error) {
      authLogger.error("Failed to fetch backend user", { error });

      // Check if this is a "user not registered" error
      if (isUserNotRegisteredError(error)) {
        const regData = getRegistrationDataFromAuth0();
        authLogger.debug("User needs registration", {
          auth0Id: regData?.auth0Id,
        });
        setNeedsRegistration(true);
        setRegistrationData(regData);
        setBackendUser(null);
        return;
      }

      setBackendError(t("errors.backendUserFetch"));
      setBackendUser(null);
    }
  }, [getAccessToken, t, getRegistrationDataFromAuth0]);

  // Retry backend fetch (for error recovery)
  const retryBackendFetch = useCallback(() => {
    if (isAuthenticated) {
      fetchBackendUser();
    }
  }, [isAuthenticated, fetchBackendUser]);

  // Register user with backend
  const register = useCallback(
    async (name: string, email: string) => {
      authLogger.debug("Starting user registration", { name });

      const api = new Api(createAuthenticatedApiConfig(getAccessToken));
      // Auth0 ID is extracted from the token by the backend middleware
      const response = await api.auth.registerCreate({
        name,
        email,
      });

      setBackendUser(response.data);
      setNeedsRegistration(false);
      setRegistrationData(null);
      authLogger.info("User registered successfully", {
        userId: response.data.id,
        name: response.data.name,
      });
    },
    [getAccessToken],
  );

  // Try cookie-based participant authentication
  // This is used when a workshop participant has accepted an invite and has a session cookie
  const tryParticipantCookieAuth = useCallback(async (): Promise<boolean> => {
    try {
      authLogger.debug("Attempting cookie-based participant authentication");
      // Use a simple fetch with credentials to check if session cookie authenticates us
      const api = new Api(getApiConfig());
      const response = await api.users.getUsers();

      if (response.data) {
        authLogger.info("Participant cookie authentication successful", {
          userId: response.data.id,
          name: response.data.name,
        });

        // Set up participant auth state
        const authUser: AuthUser = {
          sub: response.data.id || "",
          name: response.data.name || undefined,
          role: "participant",
        };
        setUser(authUser);
        setBackendUser(response.data);
        setIsAuthenticated(true);
        setIsParticipant(true);
        return true;
      }
      return false;
    } catch (error) {
      // Check for inactive workshop error
      const errorCode = extractRawErrorCode(error);
      if (errorCode === "auth_workshop_inactive") {
        authLogger.info("Participant's workshop is inactive");
        setIsWorkshopInactive(true);
        setIsParticipant(true); // They are a participant, just with inactive workshop
        return true; // Return true to prevent redirect to login
      }
      // Not authenticated via cookie - this is expected for most users
      authLogger.debug("No valid participant session cookie");
      return false;
    }
  }, []);

  // Handle Auth0 authentication state changes
  useEffect(() => {
    if (auth0IsLoading) {
      return;
    }

    if (auth0IsAuthenticated && auth0User) {
      const authUser: AuthUser = {
        sub: auth0User.sub!,
        name: auth0User.name || undefined,
        email: auth0User.email || undefined,
        picture: auth0User.picture || undefined,
      };
      setUser(authUser);
      setIsAuthenticated(true);
      setIsParticipant(false);
      authLogger.info("Auth0 authentication successful", {
        auth0Id: auth0User.sub,
        name: auth0User.name,
      });

      // Fetch backend user if not already attempted
      if (!backendFetchAttempted.current) {
        backendFetchAttempted.current = true;
        fetchBackendUser().finally(() => {
          setIsLoading(false);
        });
      } else {
        setIsLoading(false);
      }
    } else if (isDevMode && devTokenCache.current) {
      // Restore dev mode session from localStorage
      const cached = devTokenCache.current;
      authLogger.debug("Restoring dev mode session from localStorage", {
        userId: cached.userId,
        role: cached.role,
      });

      const authUser: AuthUser = {
        sub: cached.userId,
        name: cached.role,
        email: `${cached.role}@dev.local`,
        role: cached.role,
      };
      setUser(authUser);
      setIsAuthenticated(true);
      setIsParticipant(false);

      // Fetch backend user if not already attempted
      if (!backendFetchAttempted.current) {
        backendFetchAttempted.current = true;
        fetchBackendUser().finally(() => {
          setIsLoading(false);
        });
      } else {
        setIsLoading(false);
      }
    } else {
      // No Auth0 or dev mode auth - try cookie-based participant auth
      authLogger.debug("No token auth, trying cookie-based participant auth");

      if (!backendFetchAttempted.current) {
        backendFetchAttempted.current = true;
        tryParticipantCookieAuth().then((authenticated) => {
          if (!authenticated) {
            // No auth at all
            setUser(null);
            setBackendUser(null);
            setIsAuthenticated(false);
            setIsParticipant(false);
          }
          setIsLoading(false);
        });
      } else {
        setIsLoading(false);
      }
    }
  }, [
    auth0User,
    auth0IsAuthenticated,
    auth0IsLoading,
    fetchBackendUser,
    isDevMode,
    tryParticipantCookieAuth,
  ]);

  const loginWithAuth0 = () => {
    authLogger.debug("Initiating Auth0 login");
    auth0LoginWithRedirect();
  };

  const loginWithRole = async (role: string) => {
    if (!isDevMode) {
      authLogger.warning("loginWithRole called but not in dev mode");
      return;
    }

    // Well-known UUIDs for dev users (must match backend preseed.go)
    const DEV_USER_IDS: Record<string, string> = {
      admin: "00000000-0000-0000-0000-000000000001",
      head: "00000000-0000-0000-0000-000000000002",
      staff: "00000000-0000-0000-0000-000000000003",
      participant: "00000000-0000-0000-0000-000000000004",
      guest: "00000000-0000-0000-0000-000000000005",
    };

    const targetUserId = DEV_USER_IDS[role];
    if (!targetUserId) {
      authLogger.error("Unknown dev role", { role });
      setBackendError(`Unknown role: ${role}`);
      return;
    }

    authLogger.debug("Initiating dev mode login", {
      role,
      userId: targetUserId,
    });
    setIsLoading(true);
    setBackendError(null);

    try {
      const publicApi = new Api(getApiConfig());

      // Get JWT for the well-known dev user
      const jwtResponse = await publicApi.users.getUsers2(targetUserId);
      const token = jwtResponse.data.token;
      const userId = jwtResponse.data.userId;

      if (!token || !userId) {
        throw new Error(
          "Failed to get dev JWT token. Make sure the backend is running and seeded.",
        );
      }

      // Cache the token (expires in 24h according to backend)
      const tokenCache: TokenCache = {
        token,
        expiresAt: Date.now() + 23 * 60 * 60 * 1000, // 23 hours to be safe
        userId,
        role,
      };
      devTokenCache.current = tokenCache;
      storeDevToken(tokenCache);

      authLogger.info("Dev JWT obtained", { userId, role });

      // Set user state
      const authUser: AuthUser = {
        sub: userId,
        name: role,
        email: `${role}@dev.local`,
        role,
      };
      setUser(authUser);
      setIsAuthenticated(true);
      authLogger.info("Dev login successful", { role, userId });

      // Now fetch the backend user with the new token
      backendFetchAttempted.current = true;
      await fetchBackendUser();
    } catch (error) {
      authLogger.error("Dev login failed", { role, error });
      const errorMessage =
        error instanceof Error ? error.message : "Dev login failed";
      setBackendError(errorMessage);
      setUser(null);
      setIsAuthenticated(false);
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    authLogger.debug("Initiating logout", {
      isAuth0Authenticated: auth0IsAuthenticated,
    });

    // Reset backend fetch tracking
    backendFetchAttempted.current = false;
    setBackendUser(null);
    setBackendError(null);
    setNeedsRegistration(false);
    setRegistrationData(null);

    if (auth0IsAuthenticated) {
      authLogger.debug("Logging out from Auth0");
      auth0Logout({
        logoutParams: {
          returnTo: auth0Config.logoutUri,
        },
      });
    } else if (isParticipant) {
      authLogger.debug("Logging out participant (clearing session cookie)");
      // Clear participant state
      setUser(null);
      setBackendUser(null);
      setIsAuthenticated(false);
      setIsParticipant(false);
      setIsLoading(false);
      backendFetchAttempted.current = false;
      // Call backend to clear the session cookie
      fetch(`${config.API_BASE_URL}/auth/logout`, {
        method: "POST",
        credentials: "include",
      }).catch(() => {
        // Ignore errors - cookie might already be cleared
      });
      // Redirect to homepage
      authLogger.debug("Redirecting to homepage after participant logout", {
        path: ROUTES.HOME,
      });
      window.location.href = ROUTES.HOME;
    } else {
      authLogger.debug("Logging out from dev mode");
      // Clear dev token cache on logout
      devTokenCache.current = null;
      clearStoredDevToken();
      setUser(null);
      setIsAuthenticated(false);
      setIsParticipant(false);
      setIsLoading(false);
      // Redirect to homepage using window.location since we're outside RouterProvider
      authLogger.debug("Redirecting to homepage after logout", {
        path: ROUTES.HOME,
      });
      window.location.href = ROUTES.HOME;
    }
  };

  const value: AuthContextType = {
    user,
    backendUser,
    isLoading,
    isAuthenticated,
    isParticipant,
    needsRegistration,
    registrationData,
    backendError,
    isWorkshopInactive,
    loginWithAuth0,
    loginWithRole,
    logout,
    isDevMode,
    getAccessToken,
    retryBackendFetch,
    register,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
