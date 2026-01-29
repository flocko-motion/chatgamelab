import { config } from '../../config/env';

const API_BASE_URL = config.API_BASE_URL;

/**
 * Get the base API configuration without authentication.
 * Used for public endpoints only.
 * Includes credentials: 'include' to send cookies for participant auth.
 */
export function getApiConfig() {
  return {
    baseUrl: API_BASE_URL,
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
    baseApiParams: {
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include' as RequestCredentials,
      secure: true, // Enable securityWorker for all requests
    },
    securityWorker: async () => {
      const token = await getToken();
      if (!token) {
        throw new Error('No authentication token available');
      }
      return {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      };
    },
  };
}
