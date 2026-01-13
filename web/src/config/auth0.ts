import { config } from './env';

function getBaseUrl(): string {
  const publicUrl = config.PUBLIC_URL;
  if (publicUrl && publicUrl.startsWith('http')) {
    return publicUrl;
  }
  if (typeof window !== 'undefined') {
    return publicUrl === '/' 
      ? window.location.origin 
      : `${window.location.origin}${publicUrl}`;
  }
  return publicUrl || '/';
}

export const auth0Config = {
  domain: config.AUTH0_DOMAIN,
  clientId: config.AUTH0_CLIENT_ID,
  audience: config.AUTH0_AUDIENCE,
  redirectUri: config.AUTH0_REDIRECT_URI || `${getBaseUrl()}/auth/login/auth0/callback`,
  logoutUri: `${getBaseUrl()}/auth/logout/auth0/callback`,
};
