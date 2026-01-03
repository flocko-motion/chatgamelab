import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import { useAuth0 } from '@auth0/auth0-react';
import { Center, Loader, Text } from '@mantine/core';

export const Route = createFileRoute('/auth/login/auth0/callback')({
  component: Auth0Callback,
});

function Auth0Callback() {
  const { handleRedirectCallback, isLoading, error, isAuthenticated } = useAuth0();
  const navigate = useNavigate();

  useEffect(() => {
    const handleCallback = async () => {
      try {
        // Check if there are any URL parameters that indicate this is a callback
        const urlParams = new URLSearchParams(window.location.search);
        const hasCode = urlParams.has('code');
        const hasState = urlParams.has('state');
        const hasError = urlParams.has('error');
        
        console.log('Auth0 callback params:', { hasCode, hasState, hasError, isAuthenticated });
        
        if (hasError) {
          const errorMessage = urlParams.get('error_description') || 'Authentication failed';
          throw new Error(errorMessage);
        }
        
        // If we have auth parameters, handle the callback
        if (hasCode || hasState) {
          await handleRedirectCallback();
        }
        
        // If already authenticated or no auth params needed, redirect to home
        if (isAuthenticated || (!hasCode && !hasState && !hasError)) {
          navigate({ to: '/' });
          return;
        }
        
        // If we got here without being authenticated, redirect to login
        navigate({ to: '/auth/login' });
      } catch (err) {
        console.error('Auth0 callback error:', err);
        // If there's an error, redirect to login page
        navigate({ to: '/auth/login' });
      }
    };

    if (!isLoading) {
      handleCallback();
    }
  }, [handleRedirectCallback, isLoading, isAuthenticated, navigate]);

  if (isLoading) {
    return (
      <Center h="100vh">
        <div style={{ textAlign: 'center' }}>
          <Loader size="lg" mb="md" />
          <Text size="lg">Completing authentication...</Text>
        </div>
      </Center>
    );
  }

  if (error) {
    return (
      <Center h="100vh">
        <div style={{ textAlign: 'center' }}>
          <Text size="lg" c="red" mb="md">
            Authentication failed
          </Text>
          <Text size="sm" c="dimmed">
            {error.message}
          </Text>
        </div>
      </Center>
    );
  }

  return (
    <Center h="100vh">
      <Loader size="lg" />
    </Center>
  );
}
