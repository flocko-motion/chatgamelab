import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import { Center, Loader, Text } from '@mantine/core';
import { ROUTES } from '@/common/routes/routes';
import { authLogger } from '@/config/logger';

export const Route = createFileRoute('/auth/logout/auth0/callback')({
  component: Auth0LogoutCallback,
});

function Auth0LogoutCallback() {
  const navigate = useNavigate();

  useEffect(() => {
    const handleLogoutCallback = () => {
      try {
        authLogger.debug('Processing logout callback completion');
        
        // Auth0 has already logged out the user server-side
        // Just redirect to home page
        navigate({ to: ROUTES.HOME });
      } catch (err) {
        authLogger.error('Auth0 logout callback error', { error: err });
        // Even if there's an error, redirect to home
        navigate({ to: ROUTES.HOME });
      }
    };

    // Small delay to ensure the page loads properly
    const timer = setTimeout(handleLogoutCallback, 100);
    
    return () => clearTimeout(timer);
  }, [navigate]);

  return (
    <Center h="100vh">
      <div style={{ textAlign: 'center' }}>
        <Loader size="lg" mb="md" />
        <Text size="lg">Completing logout...</Text>
      </div>
    </Center>
  );
}
