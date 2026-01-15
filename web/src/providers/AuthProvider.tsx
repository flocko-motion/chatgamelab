/* eslint-disable react-refresh/only-export-components */
import React, { createContext, useCallback, useEffect, useRef, useState } from 'react';
import { useAuth0 } from '@auth0/auth0-react';
import { useTranslation } from 'react-i18next';
import { auth0Config } from '../config/auth0';
import { Api } from '../api/generated';
import { createAuthenticatedApiConfig, getApiConfig } from '../api/client/http';
import { authLogger } from '../config/logger';
import { ROUTES } from '../common/routes/routes';
import type { ObjUser } from '../api/generated';

export interface AuthUser {
  sub: string;
  name?: string;
  email?: string;
  picture?: string;
  role?: string;
}

/** Data needed for registration when user is authenticated but not registered */
export interface RegistrationData {
  auth0Id: string;
  email: string;
  name: string;
}

export interface AuthContextType {
  /** Auth0 user info (from token) */
  user: AuthUser | null;
  /** Backend user data (from /api/users/me) */
  backendUser: ObjUser | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  /** True if user is authenticated with Auth0 but not registered in backend */
  needsRegistration: boolean;
  /** Data from Auth0 to pre-fill registration form */
  registrationData: RegistrationData | null;
  /** Error fetching backend user - app is not operational */
  backendError: string | null;
  loginWithAuth0: () => void;
  loginWithRole: (role: string) => void;
  logout: () => void;
  isDevMode: boolean;
  /** Get the current access token for API calls. Returns null if not authenticated. */
  getAccessToken: () => Promise<string | null>;
  /** Retry fetching backend user after an error */
  retryBackendFetch: () => void;
  /** Register the user with the backend */
  register: (name: string, email: string) => Promise<void>;
}

export const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Default context value for error boundary recovery scenarios
const defaultAuthContext: AuthContextType = {
  user: null,
  backendUser: null,
  isLoading: true,
  isAuthenticated: false,
  needsRegistration: false,
  registrationData: null,
  backendError: null,
  loginWithAuth0: () => {},
  loginWithRole: () => {},
  logout: () => {},
  isDevMode: false,
  getAccessToken: async () => null,
  retryBackendFetch: () => {},
  register: async () => {},
};

export function useAuth() {
  const context = React.useContext(AuthContext);
  // Return default context during error boundary recovery to prevent cascading errors
  // This can happen when TanStack Router's CatchBoundaryImpl tries to re-render
  if (!context) {
    authLogger.warning('useAuth called outside AuthProvider context, returning default (error boundary recovery)');
    return defaultAuthContext;
  }
  return context;
}

interface AuthProviderProps {
  children: React.ReactNode;
}

// Token cache for dev mode
interface TokenCache {
  token: string;
  expiresAt: number;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const { t } = useTranslation();
  const {
    user: auth0User,
    isAuthenticated: auth0IsAuthenticated,
    isLoading: auth0IsLoading,
    loginWithRedirect: auth0LoginWithRedirect,
    logout: auth0Logout,
    getAccessTokenSilently,
  } = useAuth0();

  const [user, setUser] = useState<AuthUser | null>(null);
  const [backendUser, setBackendUser] = useState<ObjUser | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [backendError, setBackendError] = useState<string | null>(null);
  const [needsRegistration, setNeedsRegistration] = useState(false);
  const [registrationData, setRegistrationData] = useState<RegistrationData | null>(null);
  const [isDevMode] = useState(import.meta.env.VITE_DEV_MODE === 'true' || import.meta.env.DEV);
  
  // Token cache for dev mode tokens
  const devTokenCache = useRef<TokenCache | null>(null);
  
  // Track if we've already fetched the backend user to avoid duplicate calls
  const backendFetchAttempted = useRef(false);

  // Get access token function (defined early so it can be used in effects)
  const getAccessToken = useCallback(async (): Promise<string | null> => {
    // Auth0 authenticated - use Auth0 SDK (handles caching internally)
    if (auth0IsAuthenticated) {
      try {
        const token = await getAccessTokenSilently({
          authorizationParams: {
            audience: auth0Config.audience,
          },
        });
        return token;
      } catch (error) {
        authLogger.error('Failed to get Auth0 access token', { error });
        return null;
      }
    }

    // Dev mode - return cached token if valid
    if (devTokenCache.current && devTokenCache.current.expiresAt > Date.now()) {
      return devTokenCache.current.token;
    }

    // No valid token available
    return null;
  }, [auth0IsAuthenticated, getAccessTokenSilently]);

  // Check if error is a "user not registered" response
  const isUserNotRegisteredError = (error: unknown): boolean => {
    if (error && typeof error === 'object' && 'error' in error) {
      const errorData = (error as { error: unknown }).error;
      if (errorData && typeof errorData === 'object' && 'type' in errorData) {
        const typedError = errorData as { type: string };
        return typedError.type === 'user_not_registered';
      }
    }
    return false;
  };

  // Get registration data from Auth0 user - apply smart defaults for name
  const getRegistrationDataFromAuth0 = useCallback((): RegistrationData | null => {
    if (!auth0User?.sub) return null;
    
    const email = auth0User.email || '';
    let name = '';
    
    // Use nickname or name from Auth0, but skip if it looks like an email
    const isEmailLike = (s: string) => s.includes('@') || s.includes('+') || s === email.split('@')[0];
    
    if (auth0User.nickname && !isEmailLike(auth0User.nickname)) {
      name = auth0User.nickname;
    } else if (auth0User.name && !isEmailLike(auth0User.name)) {
      name = auth0User.name;
    }
    
    return {
      auth0Id: auth0User.sub,
      email,
      name,
    };
  }, [auth0User]);

  // Fetch backend user
  const fetchBackendUser = useCallback(async () => {
    try {
      setBackendError(null);
      setNeedsRegistration(false);
      setRegistrationData(null);
      const api = new Api(createAuthenticatedApiConfig(getAccessToken));
      const response = await api.users.getUsers();
      setBackendUser(response.data);
      authLogger.debug('Backend user fetched', { userId: response.data.id, name: response.data.name });
    } catch (error) {
      authLogger.error('Failed to fetch backend user', { error });
      
      // Check if this is a "user not registered" error
      if (isUserNotRegisteredError(error)) {
        const regData = getRegistrationDataFromAuth0();
        authLogger.debug('User needs registration', { auth0Id: regData?.auth0Id });
        setNeedsRegistration(true);
        setRegistrationData(regData);
        setBackendUser(null);
        return;
      }
      
      setBackendError(t('errors.backendUserFetch'));
      setBackendUser(null);
    }
  }, [getAccessToken, t, getRegistrationDataFromAuth0]);

  // Retry backend fetch (for error recovery)
  const retryBackendFetch = useCallback(() => {
    if (isAuthenticated) {
      fetchBackendUser();
    }
  }, [isAuthenticated, fetchBackendUser]);

  // Register user with backend
  const register = useCallback(async (name: string, email: string) => {
    authLogger.debug('Starting user registration', { name });
    
    const api = new Api(createAuthenticatedApiConfig(getAccessToken));
    // Auth0 ID is extracted from the token by the backend middleware
    const response = await api.auth.registerCreate({
      name,
      email,
    });
    
    setBackendUser(response.data);
    setNeedsRegistration(false);
    setRegistrationData(null);
    authLogger.info('User registered successfully', { userId: response.data.id, name: response.data.name });
  }, [getAccessToken]);

  // Handle Auth0 authentication state changes
  useEffect(() => {
    if (auth0IsLoading) {
      return;
    }

    if (auth0IsAuthenticated && auth0User) {
      const authUser: AuthUser = {
        sub: auth0User.sub!,
        name: auth0User.name || undefined,
        email: auth0User.email || undefined,
        picture: auth0User.picture || undefined,
      };
      setUser(authUser);
      setIsAuthenticated(true);
      authLogger.info('Auth0 authentication successful', { auth0Id: auth0User.sub, name: auth0User.name });
      
      // Fetch backend user if not already attempted
      if (!backendFetchAttempted.current) {
        backendFetchAttempted.current = true;
        fetchBackendUser().finally(() => {
          setIsLoading(false);
        });
      } else {
        setIsLoading(false);
      }
    } else {
      authLogger.debug('Auth0 authentication cleared');
      setUser(null);
      setBackendUser(null);
      setIsAuthenticated(false);
      setIsLoading(false);
      backendFetchAttempted.current = false;
    }
  }, [auth0User, auth0IsAuthenticated, auth0IsLoading, fetchBackendUser]);

  const loginWithAuth0 = () => {
    authLogger.debug('Initiating Auth0 login');
    auth0LoginWithRedirect();
  };

  const loginWithRole = async (role: string) => {
    if (!isDevMode) {
      authLogger.warning('loginWithRole called but not in dev mode');
      return;
    }

    authLogger.debug('Initiating dev mode login', { role });
    setIsLoading(true);
    setBackendError(null);

    try {
      // Use unauthenticated API to get users list
      const publicApi = new Api(getApiConfig());
      
      // Try to find an existing user or create one
      // First, try to create a dev user (will fail if already exists, that's ok)
      let targetUserId: string | undefined;
      
      try {
        const createResponse = await publicApi.users.postUsers({
          name: `Dev ${role.charAt(0).toUpperCase() + role.slice(1)}`,
          email: `${role}@dev.local`,
        });
        targetUserId = createResponse.data.id;
        authLogger.info('Created dev user', { userId: targetUserId, role });
      } catch {
        // User might already exist, try to find them
        authLogger.debug('Could not create dev user, will try to find existing', { role });
      }

      // If we couldn't create, we need to get a user ID somehow
      // The /users endpoint requires auth, so we need a different approach
      // Let's try the JWT endpoint directly with a known pattern
      if (!targetUserId) {
        // For dev mode, the backend should have seed users
        // We'll need to handle this case - for now show an error
        throw new Error(`No dev user found for role '${role}'. Please seed the database with dev users.`);
      }

      // Get JWT for this user
      const jwtResponse = await publicApi.users.getUsers2(targetUserId);
      const token = jwtResponse.data.token;
      const userId = jwtResponse.data.userId;

      if (!token || !userId) {
        throw new Error('Failed to get dev JWT token');
      }

      // Cache the token (expires in 24h according to backend)
      devTokenCache.current = {
        token,
        expiresAt: Date.now() + 23 * 60 * 60 * 1000, // 23 hours to be safe
      };

      authLogger.info('Dev JWT obtained', { userId, role });

      // Set user state
      const authUser: AuthUser = {
        sub: userId,
        name: `Dev ${role.charAt(0).toUpperCase() + role.slice(1)}`,
        email: `${role}@dev.local`,
        role,
      };
      setUser(authUser);
      setIsAuthenticated(true);
      authLogger.info('Dev login successful', { role, userId });

      // Now fetch the backend user with the new token
      backendFetchAttempted.current = true;
      await fetchBackendUser();

    } catch (error) {
      authLogger.error('Dev login failed', { role, error });
      const errorMessage = error instanceof Error ? error.message : 'Dev login failed';
      setBackendError(errorMessage);
      setUser(null);
      setIsAuthenticated(false);
    } finally {
      setIsLoading(false);
    }
  };

  const logout = () => {
    authLogger.debug('Initiating logout', { isAuth0Authenticated: auth0IsAuthenticated });
    
    // Reset backend fetch tracking
    backendFetchAttempted.current = false;
    setBackendUser(null);
    setBackendError(null);
    setNeedsRegistration(false);
    setRegistrationData(null);
    
    if (auth0IsAuthenticated) {
      authLogger.debug('Logging out from Auth0');
      auth0Logout({ 
        logoutParams: { 
          returnTo: auth0Config.logoutUri
        } 
      });
    } else {
      authLogger.debug('Logging out from dev mode');
      // Clear dev token cache on logout
      devTokenCache.current = null;
      setUser(null);
      setIsAuthenticated(false);
      setIsLoading(false);
      // Redirect to homepage using window.location since we're outside RouterProvider
      authLogger.debug('Redirecting to homepage after logout', { path: ROUTES.HOME });
      window.location.href = ROUTES.HOME;
    }
  };

  const value: AuthContextType = {
    user,
    backendUser,
    isLoading,
    isAuthenticated,
    needsRegistration,
    registrationData,
    backendError,
    loginWithAuth0,
    loginWithRole,
    logout,
    isDevMode,
    getAccessToken,
    retryBackendFetch,
    register,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
