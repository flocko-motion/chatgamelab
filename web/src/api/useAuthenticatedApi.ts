import { useMemo } from 'react';
import { useAuth } from '@/providers/AuthProvider';
import { Api } from './generated';
import { createAuthenticatedApiConfig } from './client/http';

/**
 * Hook that provides an authenticated API client.
 * The client automatically injects the Bearer token on all requests.
 * 
 * @throws Error if user is not authenticated and a request is made
 */
export function useAuthenticatedApi() {
  const { getAccessToken, isAuthenticated } = useAuth();

  const api = useMemo(() => {
    if (!isAuthenticated) {
      return null;
    }
    return new Api(createAuthenticatedApiConfig(getAccessToken));
  }, [getAccessToken, isAuthenticated]);

  return api;
}

/**
 * Hook that provides an authenticated API client, throwing if not authenticated.
 * Use this in components that are guaranteed to be rendered only when authenticated.
 */
export function useRequiredAuthenticatedApi() {
  const api = useAuthenticatedApi();
  
  if (!api) {
    throw new Error('useRequiredAuthenticatedApi must be used when user is authenticated');
  }
  
  return api;
}
