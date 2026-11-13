import { config } from '../../config/env';

const API_BASE_URL = config.API_BASE_URL;

/** Fired on window when the server says the user's organisation is locked. */
export const INSTITUTION_DISABLED_EVENT = 'cgl:institution-disabled';

/**
 * fetch that watches for "organisation locked" answers. Pages that were already open when an
 * admin locked the organisation would otherwise only show generic load errors; the event lets
 * the auth layer reload the user and show the lock notice instead.
 */
const watchingFetch: typeof fetch = async (...args) => {
  const response = await fetch(...args);
  if (response.status === 403) {
    response
      .clone()
      .json()
      .then((body: { code?: string }) => {
        if (body?.code === 'institution_disabled') {
          window.dispatchEvent(new Event(INSTITUTION_DISABLED_EVENT));
        }
      })
      .catch(() => {});
  }
  return response;
};

/**
 * Get the base API configuration without authentication.
 * Used for public endpoints only.
 * Includes credentials: 'include' to send cookies for participant auth.
 */
export function getApiConfig() {
  return {
    baseUrl: API_BASE_URL,
    customFetch: watchingFetch,
    baseApiParams: {
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include' as RequestCredentials,
    },
  };
}

/**
 * Create an authenticated API configuration.
 * Uses a token getter function to inject the Authorization header on each request.
 * Includes credentials: 'include' to send cookies for participant auth fallback.
 */
export function createAuthenticatedApiConfig(getToken: () => Promise<string | null>) {
  return {
    baseUrl: API_BASE_URL,
    customFetch: watchingFetch,
    baseApiParams: {
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include' as RequestCredentials,
      secure: true, // Enable securityWorker for all requests
    },
    securityWorker: async () => {
      const token = await getToken();
      // If no token, rely on cookie authentication (for participants)
      if (!token) {
        return {};
      }
      return {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      };
    },
  };
}
