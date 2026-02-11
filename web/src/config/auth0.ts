import { config } from './env';
import { getBaseUrl } from '@/common/lib/url';

export const auth0Config = {
  domain: config.AUTH0_DOMAIN,
  clientId: config.AUTH0_CLIENT_ID,
  audience: config.AUTH0_AUDIENCE,
  redirectUri: config.AUTH0_REDIRECT_URI || `${getBaseUrl()}/auth/login/auth0/callback`,
  logoutUri: `${getBaseUrl()}/auth/logout/auth0/callback`,
};
