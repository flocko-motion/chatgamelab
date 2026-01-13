import { config } from './env';

export const auth0Config = {
  domain: config.AUTH0_DOMAIN,
  clientId: config.AUTH0_CLIENT_ID,
  audience: config.AUTH0_AUDIENCE,
  redirectUri: config.AUTH0_REDIRECT_URI || `${config.PUBLIC_URL}/auth/login/auth0/callback`,
  logoutUri: `${config.PUBLIC_URL}/auth/logout/auth0/callback`,
};
