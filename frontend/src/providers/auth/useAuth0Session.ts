/**
 * Auth0 sign-in, when the instance offers it: signs in with the Auth0 SDK and
 * exchanges the ID token for a schemaf session (POST /api/auth/auth0/login).
 * From then on the session cookie authenticates every call.
 */
import { useCallback } from "react";
import { useAuth0 } from "@auth0/auth0-react";
import { useTranslation } from "react-i18next";
import { api } from "@/api/client/api";
import { auth0Config } from "@/config/auth0";
import { authLogger } from "@/config/logger";

const AUTH0_LOCALE_MAP: Record<string, string> = { de: "de", en: "en", fr: "fr-FR", es: "es", it: "it" };

// schemaf refuses ID tokens issued more than five minutes ago.
const MAX_ID_TOKEN_AGE_SECONDS = 4 * 60;

export function useAuth0Session() {
  const { i18n } = useTranslation();
  const auth0 = useAuth0();
  const enabled = auth0Config !== null;
  const { getIdTokenClaims, getAccessTokenSilently, loginWithRedirect, logout } = auth0;

  /**
   * Exchanges the Auth0 sign-in for a schemaf session. Returns false if there
   * is no usable Auth0 sign-in; the SDK's local state is then cleared.
   */
  const exchange = useCallback(async (): Promise<boolean> => {
    try {
      let claims = await getIdTokenClaims();
      const age = claims?.iat ? Date.now() / 1000 - claims.iat : Infinity;
      if (age > MAX_ID_TOKEN_AGE_SECONDS) {
        await getAccessTokenSilently({ cacheMode: "off" });
        claims = await getIdTokenClaims();
      }
      if (!claims?.__raw) return false;
      await api.postApiAuthAuth0Login({ id_token: claims.__raw });
      authLogger.info("Auth0 sign-in exchanged for a session");
      return true;
    } catch (error) {
      authLogger.warning("Auth0 sign-in could not be exchanged", { error });
      await logout({ openUrl: false });
      return false;
    }
  }, [getIdTokenClaims, getAccessTokenSilently, logout]);

  const login = useCallback(
    (options?: { screenHint?: string }) => {
      if (!enabled) return;
      const auth0Locale = AUTH0_LOCALE_MAP[i18n.language?.split("-")[0]];
      void loginWithRedirect({
        authorizationParams: {
          ui_locales: auth0Locale ? `${auth0Locale} en` : "en",
          ...(options?.screenHint && { screen_hint: options.screenHint }),
        },
      });
    },
    [enabled, i18n.language, loginWithRedirect],
  );

  /** Signs out at Auth0 too; the browser leaves for Auth0 and comes back. */
  const logoutAtAuth0 = useCallback(async () => {
    if (!auth0Config) return;
    await logout({ logoutParams: { returnTo: auth0Config.logoutUri } });
  }, [logout]);

  return {
    enabled,
    // Without Auth0Provider the SDK reports loading forever.
    isLoading: enabled && auth0.isLoading,
    isAuthenticated: enabled && auth0.isAuthenticated,
    auth0User: enabled ? auth0.user : undefined,
    exchange,
    login,
    logoutAtAuth0,
  };
}
