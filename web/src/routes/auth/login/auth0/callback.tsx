import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect, useRef } from 'react';
import { useAuth0 } from '@auth0/auth0-react';
import { Center, Loader, Text, Button, Stack } from '@mantine/core';
import { ROUTES } from '@/common/routes/routes';
import { authLogger } from '@/config/logger';

export const Route = createFileRoute('/auth/login/auth0/callback')({
  component: Auth0Callback,
});

function Auth0Callback() {
  const { isLoading, error, isAuthenticated } = useAuth0();
  const navigate = useNavigate();
  const hasNavigated = useRef(false);

  useEffect(() => {
    if (isLoading || hasNavigated.current) return;

    // Check for error params in URL (Auth0 may redirect back with error)
    const urlParams = new URLSearchParams(window.location.search);
    const urlError = urlParams.get('error');
    const urlErrorDesc = urlParams.get('error_description');

    if (urlError) {
      authLogger.error('Auth0 callback URL error', { error: urlError, description: urlErrorDesc });
      // Don't redirect to login — that causes an infinite loop in production.
      // The error UI below will render instead.
      return;
    }

    if (isAuthenticated) {
      hasNavigated.current = true;
      authLogger.debug('Auth0 callback: authenticated, redirecting to dashboard');
      navigate({ to: ROUTES.DASHBOARD });
    }
    // If not authenticated and not loading, Auth0Provider is still processing
    // the callback (exchanging code for tokens). Wait for the next render.
  }, [isLoading, isAuthenticated, navigate]);

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

  // Show error from Auth0 SDK or from URL params
  const urlParams = new URLSearchParams(window.location.search);
  const urlError = urlParams.get('error_description') || urlParams.get('error');
  const displayError = error?.message || urlError;

  if (displayError) {
    return (
      <Center h="100vh">
        <Stack align="center" gap="md">
          <Text size="lg" c="red">
            Authentication failed
          </Text>
          <Text size="sm" c="dimmed" maw={400} ta="center">
            {displayError}
          </Text>
          <Button
            variant="outline"
            onClick={() => navigate({ to: ROUTES.AUTH_LOGIN })}
          >
            Try again
          </Button>
        </Stack>
      </Center>
    );
  }

  return (
    <Center h="100vh">
      <Loader size="lg" />
    </Center>
  );
}
