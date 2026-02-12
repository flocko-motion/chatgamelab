/**
 * Hook that manages fetching and caching the backend user profile.
 * Handles registration detection, language sync, and error recovery.
 */
import { useCallback, useRef, useState } from "react";
import { useAuth0 } from "@auth0/auth0-react";
import { Api } from "@/api/generated";
import { createAuthenticatedApiConfig } from "@/api/client/http";
import { authLogger } from "@/config/logger";
import { ErrorCodes, extractErrorCode } from "@/common/types/errorCodes";
import type { ObjUser } from "@/api/generated";
import i18n from "@/i18n";
import type { RegistrationData } from "./types";

interface UseBackendUserOptions {
  getAccessToken: () => Promise<string | null>;
}

export function useBackendUser({ getAccessToken }: UseBackendUserOptions) {
  const { user: auth0User } = useAuth0();

  const [backendUser, setBackendUser] = useState<ObjUser | null>(null);
  const [backendError, setBackendError] = useState<string | null>(null);
  const [needsRegistration, setNeedsRegistration] = useState(false);
  const [registrationData, setRegistrationData] =
    useState<RegistrationData | null>(null);

  // Ref for t to avoid fetchBackendUser identity changes when i18next re-renders
  const tRef = useRef<(key: string) => string>((key) => key);

  /** Update the translation function ref (call from provider render). */
  const setTranslationRef = (t: (key: string) => string) => {
    tRef.current = t;
  };

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

  // Fetch backend user. Returns true on success, false on failure.
  const fetchBackendUser = useCallback(async (): Promise<boolean> => {
    try {
      setBackendError(null);
      setNeedsRegistration(false);
      setRegistrationData(null);
      const api = new Api(createAuthenticatedApiConfig(getAccessToken));
      const response = await api.users.getUsers();
      setBackendUser(response.data);
      // Apply user's stored language preference on login
      if (response.data.language && response.data.language !== i18n.language) {
        authLogger.debug("Applying user language preference", {
          stored: response.data.language,
          current: i18n.language,
        });
        i18n.changeLanguage(response.data.language);
      }
      authLogger.debug("Backend user fetched", {
        userId: response.data.id,
        name: response.data.name,
      });
      return true;
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
        return false;
      }

      setBackendError(tRef.current("errors.backendUserFetch"));
      setBackendUser(null);
      return false;
    }
  }, [getAccessToken, getRegistrationDataFromAuth0]);

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

  return {
    backendUser,
    setBackendUser,
    backendError,
    setBackendError,
    needsRegistration,
    registrationData,
    fetchBackendUser,
    register,
    setTranslationRef,
  };
}
