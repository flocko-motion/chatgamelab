import { config } from '../../config/env';

const API_BASE_URL = config.API_BASE_URL;

/**
 * Get the base API configuration without authentication.
 * Used for public endpoints only.
 */
export function getApiConfig() {
  return {
    baseUrl: API_BASE_URL,
    baseApiParams: {
      headers: {
        'Content-Type': 'application/json',
      },
    },
  };
}

/**
 * Create an authenticated API configuration.
 * Uses a token getter function to inject the Authorization header on each request.
 */
export function createAuthenticatedApiConfig(getToken: () => Promise<string | null>) {
  return {
    baseUrl: API_BASE_URL,
    baseApiParams: {
      headers: {
        'Content-Type': 'application/json',
      },
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
