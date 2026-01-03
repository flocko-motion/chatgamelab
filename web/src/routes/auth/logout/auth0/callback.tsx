import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import { useAuth0 } from '@auth0/auth0-react';
import { Center, Loader, Text } from '@mantine/core';

export const Route = createFileRoute('/auth/logout/auth0/callback')({
  component: Auth0LogoutCallback,
});

function Auth0LogoutCallback() {
  const { isAuthenticated } = useAuth0();
  const navigate = useNavigate();

  useEffect(() => {
    const handleLogoutCallback = () => {
      try {
        // At this point, Auth0 has already logged out the user
        // We just need to clear any local state and redirect
        console.log('Logout callback: user is now', isAuthenticated ? 'authenticated' : 'not authenticated');
        
        // Redirect to home page after logout
        navigate({ to: '/' });
      } catch (err) {
        console.error('Auth0 logout callback error:', err);
        // Even if there's an error, redirect to home
        navigate({ to: '/' });
      }
    };

    // Small delay to ensure the page loads properly
    const timer = setTimeout(handleLogoutCallback, 100);
    
    return () => clearTimeout(timer);
  }, [isAuthenticated, navigate]);

  return (
    <Center h="100vh">
      <div style={{ textAlign: 'center' }}>
        <Loader size="lg" mb="md" />
        <Text size="lg">Completing logout...</Text>
      </div>
    </Center>
  );
}
