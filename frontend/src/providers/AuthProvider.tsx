/* eslint-disable react-refresh/only-export-components */
import React, { createContext, useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "@/api/client/api";
import { authLogger } from "../config/logger";
import { getHomepageUrl } from "../common/lib/url";
import { RoleName } from "../common/lib/roles";
import type { AuthContextType, AuthUser } from "./auth/types";
import { useCaller } from "./auth/useCaller";
import { useAuth0Session } from "./auth/useAuth0Session";

export type { AuthUser, RegistrationData, AuthContextType } from "./auth/types";
export type { CallerStanding, Registration } from "./auth/useCaller";
export {
  guestLogin,
  rememberGuestCode,
  rememberedGuestCode,
  type GuestLoginResult,
} from "./auth/guestLogin";
import { forgetGuestCode } from "./auth/guestLogin";

// The preset dev accounts sign in only on the dev server; the backend refuses
// them outside development mode too.
const isDevMode = import.meta.env.DEV;

export const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Default context value for error boundary recovery scenarios
const defaultAuthContext: AuthContextType = {
  user: null,
  backendUser: null,
  isLoading: true,
  isAuthenticated: false,
  isGuest: false,
  needsRegistration: false,
  registrationData: null,
  backendError: null,
  isWorkshopInactive: false,
  auth0Enabled: false,
  loginWithAuth0: () => {},
  isDevMode: false,
  loginWithRole: async () => {},
  logout: () => {},
  retryBackendFetch: () => {},
  register: async () => {},
  standing: { mayConsentForInstitution: false, maySuspendInstitutions: false },
  refreshCaller: async () => {},
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

/** A name to suggest at registration: Auth0's nickname or name, unless it looks like an email. */
function suggestedName(auth0User: { nickname?: string; name?: string; email?: string } | undefined): string {
  if (!auth0User) return "";
  const local = auth0User.email?.split("@")[0];
  const looksLikeEmail = (s: string) => s.includes("@") || s.includes("+") || s === local;
  for (const candidate of [auth0User.nickname, auth0User.name]) {
    if (candidate && !looksLikeEmail(candidate)) return candidate;
  }
  return "";
}

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { t } = useTranslation();
  const caller = useCaller();
  const auth0 = useAuth0Session();
  caller.setTranslationRef(t);

  const [settled, setSettled] = useState(false);
  const started = useRef(false);
  const exchanged = useRef(false);

  const { fetchCaller } = caller;
  const { exchange } = auth0;

  // Who is calling: the session cookie decides. A pending Auth0 sign-in is
  // exchanged for a session first.
  useEffect(() => {
    if (auth0.isLoading || started.current) return;
    started.current = true;
    void (async () => {
      const state = await fetchCaller();
      if (state === "anonymous" && auth0.isAuthenticated && !exchanged.current) {
        exchanged.current = true;
        if (await exchange()) await fetchCaller();
      }
      setSettled(true);
    })();
  }, [auth0.isLoading, auth0.isAuthenticated, fetchCaller, exchange]);

  const loginWithRole = async (role: string) => {
    if (!isDevMode) return;
    setSettled(false);
    try {
      await api.postApiAuthDevLogin({ role });
      authLogger.info("Dev sign-in", { role });
    } catch (error) {
      authLogger.error("Dev sign-in failed", { role, error });
    }
    await fetchCaller();
    setSettled(true);
  };

  const retryBackendFetch = useCallback(() => {
    void fetchCaller();
  }, [fetchCaller]);

  const refreshCaller = useCallback(async () => {
    await fetchCaller();
  }, [fetchCaller]);

  const logout = async () => {
    authLogger.debug("Signing out");
    // Keep loading, so no route redirects before the browser leaves.
    setSettled(false);
    try {
      await api.postApiAuthLogout();
    } catch (error) {
      authLogger.debug("Logout request failed", { error });
    }
    caller.clear();
    forgetGuestCode();
    if (auth0.isAuthenticated) {
      await auth0.logoutAtAuth0();
      return;
    }
    window.location.href = getHomepageUrl();
  };

  const backendUser = caller.user;
  const user: AuthUser | null = backendUser
    ? {
        sub: backendUser.id,
        name: backendUser.name || undefined,
        email: backendUser.email || undefined,
        role: backendUser.role?.role,
      }
    : null;
  const needsRegistration = caller.state === "registering";

  const value: AuthContextType = {
    user,
    backendUser,
    isLoading: !settled,
    isAuthenticated: caller.state === "registered",
    isGuest: backendUser?.role?.role === RoleName.Guest || caller.isWorkshopInactive,
    needsRegistration,
    registrationData: needsRegistration
      ? { name: suggestedName(auth0.auth0User), email: auth0.auth0User?.email ?? "" }
      : null,
    backendError: caller.error,
    isWorkshopInactive: caller.isWorkshopInactive,
    auth0Enabled: auth0.enabled,
    loginWithAuth0: auth0.login,
    isDevMode,
    loginWithRole,
    logout,
    retryBackendFetch,
    register: caller.register,
    standing: caller.standing,
    refreshCaller,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
