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
import { getApiConfig } from "../api/client/http";
import { authLogger } from "../config/logger";
import { ROUTES } from "../common/routes/routes";
import { buildShareUrl } from "../common/lib/url";
import { extractRawErrorCode } from "../common/types/errorCodes";
import type { AuthUser, AuthContextType } from "./auth/types";
import { useTokenManager } from "./auth/useTokenManager";
import { useBackendUser } from "./auth/useBackendUser";
import {
  getStoredParticipantToken,
  clearParticipantToken,
  storeDevToken,
  clearStoredDevToken,
  type TokenCache,
} from "./auth/tokenStorage";

// Re-export types and token helpers for external consumers
export type { AuthUser, RegistrationData, AuthContextType } from "./auth/types";
export {
  getStoredParticipantToken,
  storeParticipantToken,
  clearParticipantToken,
} from "./auth/tokenStorage";

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

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const { t } = useTranslation();
  const {
    user: auth0User,
    isAuthenticated: auth0IsAuthenticated,
    isLoading: auth0IsLoading,
    loginWithRedirect: auth0LoginWithRedirect,
    logout: auth0Logout,
  } = useAuth0();

  // ── Composed hooks ──────────────────────────────────────────────────

  const {
    getAccessToken,
    isParticipant,
    setIsParticipant,
    isDevMode,
    devTokenCache,
  } = useTokenManager();

  const {
    backendUser,
    setBackendUser,
    backendError,
    setBackendError,
    needsRegistration,
    registrationData,
    fetchBackendUser,
    register,
    setTranslationRef,
  } = useBackendUser({ getAccessToken });

  // Keep translation ref in sync
  setTranslationRef(t);

  // ── Local state ─────────────────────────────────────────────────────

  const [user, setUser] = useState<AuthUser | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isWorkshopInactive, setIsWorkshopInactive] = useState(false);

  // Track if we've already fetched the backend user to avoid duplicate calls
  const backendFetchAttempted = useRef(false);

  // Retry backend fetch (for error recovery)
  const retryBackendFetch = useCallback(() => {
    if (isAuthenticated) {
      fetchBackendUser();
    }
  }, [isAuthenticated, fetchBackendUser]);

  // ── Participant Auth ────────────────────────────────────────────────

  // Try participant authentication (via stored token or cookie)
  const tryParticipantAuth = useCallback(async (): Promise<boolean> => {
    const storedToken = getStoredParticipantToken();

    try {
      let api: Api<unknown>;

      if (storedToken) {
        authLogger.debug(
          "Attempting participant authentication with stored token",
        );
        api = new Api({
          baseUrl: config.API_BASE_URL,
          baseApiParams: {
            headers: {
              "Content-Type": "application/json",
              Authorization: `Bearer ${storedToken}`,
            },
            credentials: "include" as RequestCredentials,
          },
        });
      } else {
        authLogger.debug("Attempting cookie-based participant authentication");
        api = new Api(getApiConfig());
      }

      const response = await api.users.getUsers();

      if (response.data) {
        authLogger.info("Participant authentication successful", {
          userId: response.data.id,
          name: response.data.name,
          method: storedToken ? "token" : "cookie",
        });

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
      const errorCode = extractRawErrorCode(error);
      if (errorCode === "auth_workshop_inactive") {
        authLogger.info("Participant's workshop is inactive");
        setIsWorkshopInactive(true);
        setIsParticipant(true);
        return true;
      }
      if (storedToken) {
        authLogger.debug("Stored participant token invalid, clearing");
        clearParticipantToken();
      }
      authLogger.debug("No valid participant authentication");
      return false;
    }
  }, [setBackendUser, setIsParticipant]);

  // ── Auth State Effect ───────────────────────────────────────────────

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

      if (!backendFetchAttempted.current) {
        backendFetchAttempted.current = true;
        fetchBackendUser().finally(() => {
          setIsLoading(false);
        });
      } else {
        setIsLoading(false);
      }
    } else if (isDevMode && devTokenCache.current) {
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

      if (!backendFetchAttempted.current) {
        backendFetchAttempted.current = true;
        fetchBackendUser().finally(() => {
          setIsLoading(false);
        });
      } else {
        setIsLoading(false);
      }
    } else {
      authLogger.debug("No token auth, trying cookie-based participant auth");

      if (!backendFetchAttempted.current) {
        backendFetchAttempted.current = true;
        tryParticipantAuth().then((authenticated) => {
          if (!authenticated) {
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
    tryParticipantAuth,
    devTokenCache,
    setBackendUser,
    setIsParticipant,
  ]);

  // ── Login / Logout ──────────────────────────────────────────────────

  const loginWithAuth0 = () => {
    authLogger.debug("Initiating Auth0 login");
    auth0LoginWithRedirect();
  };

  const loginWithRole = async (role: string) => {
    if (!isDevMode) {
      authLogger.warning("loginWithRole called but not in dev mode");
      return;
    }

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

      const jwtResponse = await publicApi.users.getUsers2(targetUserId);
      const token = jwtResponse.data.token;
      const userId = jwtResponse.data.userId;

      if (!token || !userId) {
        throw new Error(
          "Failed to get dev JWT token. Make sure the backend is running and seeded.",
        );
      }

      const tokenCache: TokenCache = {
        token,
        expiresAt: Date.now() + 23 * 60 * 60 * 1000,
        userId,
        role,
      };
      devTokenCache.current = tokenCache;
      storeDevToken(tokenCache);

      authLogger.info("Dev JWT obtained", { userId, role });

      const authUser: AuthUser = {
        sub: userId,
        name: role,
        email: `${role}@dev.local`,
        role,
      };
      setUser(authUser);
      setIsAuthenticated(true);
      authLogger.info("Dev login successful", { role, userId });

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

  const logout = async () => {
    authLogger.debug("Initiating logout", {
      isAuth0Authenticated: auth0IsAuthenticated,
    });

    backendFetchAttempted.current = false;
    setBackendUser(null);
    setBackendError(null);

    if (auth0IsAuthenticated) {
      authLogger.debug("Logging out from Auth0");
      auth0Logout({
        logoutParams: {
          returnTo: auth0Config.logoutUri,
        },
      });
    } else if (isParticipant) {
      authLogger.debug("Logging out participant (clearing session and token)");
      setUser(null);
      setBackendUser(null);
      setIsAuthenticated(false);
      setIsParticipant(false);
      setIsLoading(false);
      backendFetchAttempted.current = false;
      clearParticipantToken();
      try {
        await fetch(`${config.API_BASE_URL}/auth/logout`, {
          method: "POST",
          credentials: "include",
        });
        authLogger.debug("Session cookie cleared by backend");
      } catch {
        authLogger.debug(
          "Failed to clear session cookie (may already be cleared)",
        );
      }
      authLogger.debug("Redirecting to homepage after participant logout", {
        path: ROUTES.HOME,
      });
      window.location.href = buildShareUrl(ROUTES.HOME);
    } else {
      authLogger.debug("Logging out from dev mode");
      devTokenCache.current = null;
      clearStoredDevToken();
      setUser(null);
      setIsAuthenticated(false);
      setIsParticipant(false);
      setIsLoading(false);
      authLogger.debug("Redirecting to homepage after logout", {
        path: ROUTES.HOME,
      });
      window.location.href = buildShareUrl(ROUTES.HOME);
    }
  };

  // ── Context Value ───────────────────────────────────────────────────

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
