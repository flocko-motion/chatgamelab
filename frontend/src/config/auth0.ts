import { config } from './env';
import { getBaseUrl } from '@/common/lib/url';

/** Auth0 sign-in settings; null when this instance offers no Auth0 sign-in. */
export const auth0Config =
  config.AUTH0_DOMAIN && config.AUTH0_CLIENT_ID
    ? {
        domain: config.AUTH0_DOMAIN,
        clientId: config.AUTH0_CLIENT_ID,
        redirectUri: `${getBaseUrl()}/auth/login/auth0/callback`,
        logoutUri: `${getBaseUrl()}/auth/logout/auth0/callback`,
      }
    : null;
