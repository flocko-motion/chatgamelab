// Mock-aware authentication wrapper
// This replaces withAuthenticationRequired to support both real Auth0 and mock authentication

import React from 'react';
import { useAuth0 } from "@auth0/auth0-react";
import { useRecoilValue } from "recoil";
import { mockAuthState, cglAuthState } from "../api/atoms";
import Loading from "../components/Loading";

export const withMockAwareAuth = (Component, options = {}) => {
  const onRedirecting = options.onRedirecting || (() => <Loading />);

  return function MockAwareAuthenticatedComponent(props) {
    const { isAuthenticated, isLoading, loginWithRedirect } = useAuth0();
    const isAuthenticatedMock = useRecoilValue(mockAuthState);
    const isAuthenticatedCgl = useRecoilValue(cglAuthState);

    // Check if user is authenticated via any method
    const actuallyAuthenticated = isAuthenticated || isAuthenticatedMock || isAuthenticatedCgl;

    // Show loading if Auth0 is still loading (but not in mock mode)
    if (isLoading && !isAuthenticatedMock) {
      return onRedirecting();
    }

    // If not authenticated by any method, redirect to login
    if (!actuallyAuthenticated) {
      // In a real app, this would trigger Auth0 redirect
      // But our route protection in App.js should prevent this from happening
      console.warn('[AUTH] Component accessed without authentication - route protection should handle this');
      loginWithRedirect();
      return onRedirecting();
    }

    // User is authenticated (either real or mock), render component
    return <Component {...props} />;
  };
};