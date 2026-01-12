import React, { createContext, useContext, useEffect, useState } from 'react';
import { useAuth0 } from '@auth0/auth0-react';

export interface AuthUser {
  sub: string;
  name?: string;
  email?: string;
  picture?: string;
  role?: string;
}

export interface AuthContextType {
  user: AuthUser | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  loginWithAuth0: () => void;
  loginWithRole: (role: string) => void;
  logout: () => void;
  isDevMode: boolean;
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

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const {
    user: auth0User,
    isAuthenticated: auth0IsAuthenticated,
    isLoading: auth0IsLoading,
    loginWithRedirect: auth0LoginWithRedirect,
    logout: auth0Logout,
  } = useAuth0();

  const [user, setUser] = useState<AuthUser | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isDevMode] = useState(import.meta.env.VITE_DEV_MODE === 'true' || import.meta.env.DEV);

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
      setIsLoading(false);
    } else {
      setUser(null);
      setIsAuthenticated(false);
      setIsLoading(false);
    }
  }, [auth0User, auth0IsAuthenticated, auth0IsLoading]);

  const loginWithAuth0 = () => {
    auth0LoginWithRedirect();
  };

  const loginWithRole = (role: string) => {
    const mockUser: AuthUser = {
      sub: `dev-${role}`,
      name: `${role.charAt(0).toUpperCase() + role.slice(1)} User`,
      email: `${role}@dev.local`,
      role,
    };
    setUser(mockUser);
    setIsAuthenticated(true);
    setIsLoading(false);
  };

  const logout = () => {
    if (auth0IsAuthenticated) {
      auth0Logout({ 
        logoutParams: { 
          returnTo: `${window.location.origin}/auth/logout/auth0/callback` 
        } 
      });
    } else {
      setUser(null);
      setIsAuthenticated(false);
      setIsLoading(false);
    }
  };

  const value: AuthContextType = {
    user,
    isLoading,
    isAuthenticated,
    loginWithAuth0,
    loginWithRole,
    logout,
    isDevMode,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
