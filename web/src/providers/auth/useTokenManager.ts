/**
 * Hook that manages access token retrieval across all auth modes:
 * Auth0, dev mode, and participant (cookie/stored token).
 */
import { useCallback, useRef, useState } from "react";
import { useAuth0 } from "@auth0/auth0-react";
import { auth0Config } from "@/config/auth0";
import { authLogger } from "@/config/logger";
import {
  getStoredParticipantToken,
  getStoredDevToken,
  type TokenCache,
} from "./tokenStorage";

export function useTokenManager() {
  const { isAuthenticated: auth0IsAuthenticated, getAccessTokenSilently } =
    useAuth0();

  const [isParticipant, setIsParticipant] = useState(false);
  const [isDevMode] = useState(
    import.meta.env.VITE_DEV_MODE === "true" || import.meta.env.DEV,
  );

  // Token cache for dev mode tokens (initialize from localStorage)
  const devTokenCache = useRef<TokenCache | null>(getStoredDevToken());

  const getAccessToken = useCallback(async (): Promise<string | null> => {
    // Participants use stored token (if available) or cookie auth
    if (isParticipant) {
      const storedToken = getStoredParticipantToken();
      return storedToken; // Returns null if no stored token, which falls back to cookie
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

  return {
    getAccessToken,
    isParticipant,
    setIsParticipant,
    isDevMode,
    devTokenCache,
  };
}
