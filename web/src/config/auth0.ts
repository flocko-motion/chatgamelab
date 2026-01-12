export const auth0Config = {
  domain: import.meta.env.VITE_AUTH0_DOMAIN,
  clientId: import.meta.env.VITE_AUTH0_CLIENT_ID,
  audience: import.meta.env.VITE_AUTH0_AUDIENCE,
  redirectUri: `${window.location.origin}/auth/login/auth0/callback`,
  logoutUri: `${window.location.origin}/auth/logout/auth0/callback`,
};
