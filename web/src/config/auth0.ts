import { config } from './env';

export const auth0Config = {
  domain: config.AUTH0_DOMAIN,
  clientId: config.AUTH0_CLIENT_ID,
  audience: config.AUTH0_AUDIENCE,
  redirectUri: config.AUTH0_REDIRECT_URI || `${window.location.origin}/auth/login/auth0/callback`,
  logoutUri: `${window.location.origin}/auth/logout/auth0/callback`,
};
