import React, { createContext, useCallback, useContext, useEffect, useRef, useState } from 'react';
import { useAuth0 } from '@auth0/auth0-react';
import { useTranslation } from 'react-i18next';
import { auth0Config } from '../config/auth0';
import { Api } from '../api/generated';
import { createAuthenticatedApiConfig } from '../api/client/http';
import type { ObjUser } from '../api/generated';

export interface AuthUser {
  sub: string;
  name?: string;
  email?: string;
  picture?: string;
  role?: string;
}

export interface AuthContextType {
  /** Auth0 user info (from token) */
  user: AuthUser | null;
  /** Backend user data (from /api/users/me) */
  backendUser: ObjUser | null;
  isLoading: boolean;
  isAuthenticated: boolean;
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
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

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
        console.error('[Auth] Failed to get Auth0 access token:', error);
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

  // Fetch backend user
  const fetchBackendUser = useCallback(async () => {
    try {
      setBackendError(null);
      const api = new Api(createAuthenticatedApiConfig(getAccessToken));
      const response = await api.users.getUsers();
      setBackendUser(response.data);
      console.log('[Auth] Backend user fetched:', response.data.name);
    } catch (error) {
      console.error('[Auth] Failed to fetch backend user:', error);
      setBackendError(t('errors.backendUserFetch'));
      setBackendUser(null);
    }
  }, [getAccessToken, t]);

  // Retry backend fetch (for error recovery)
  const retryBackendFetch = useCallback(() => {
    if (isAuthenticated) {
      fetchBackendUser();
    }
  }, [isAuthenticated, fetchBackendUser]);

  // Handle Auth0 authentication state changes
  useEffect(() => {
    if (auth0IsLoading) {
      setIsLoading(true);
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
      setUser(null);
      setBackendUser(null);
      setIsAuthenticated(false);
      setIsLoading(false);
      backendFetchAttempted.current = false;
    }
  }, [auth0User, auth0IsAuthenticated, auth0IsLoading, fetchBackendUser]);

  const loginWithAuth0 = () => {
    auth0LoginWithRedirect();
  };

  const loginWithRole = async (role: string) => {
    const mockUser: AuthUser = {
      sub: `dev-${role}`,
      name: `${role.charAt(0).toUpperCase() + role.slice(1)} User`,
      email: `${role}@dev.local`,
      role,
    };
    setUser(mockUser);
    setIsAuthenticated(true);
    
    // For dev mode, we need to fetch a dev JWT and then the backend user
    // TODO: Implement dev mode backend integration
    setIsLoading(false);
  };

  const logout = () => {
    // Reset backend fetch tracking
    backendFetchAttempted.current = false;
    setBackendUser(null);
    setBackendError(null);
    
    if (auth0IsAuthenticated) {
      auth0Logout({ 
        logoutParams: { 
          returnTo: `${window.location.origin}/auth/logout/auth0/callback` 
        } 
      });
    } else {
      // Clear dev token cache on logout
      devTokenCache.current = null;
      setUser(null);
      setIsAuthenticated(false);
      setIsLoading(false);
    }
  };

  const value: AuthContextType = {
    user,
    backendUser,
    isLoading,
    isAuthenticated,
    backendError,
    loginWithAuth0,
    loginWithRole,
    logout,
    isDevMode,
    getAccessToken,
    retryBackendFetch,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
