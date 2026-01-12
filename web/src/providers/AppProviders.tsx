import { MantineProvider } from '@mantine/core';
import { ModalsProvider } from '@mantine/modals';
import { Notifications } from '@mantine/notifications';
import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { RouterProvider } from '@tanstack/react-router';
import { Auth0Provider } from '@auth0/auth0-react';
import { ErrorBoundary } from '@/common/components/ErrorBoundary';

import { mantineTheme } from '../config/mantineTheme';
import { queryClient } from '../config/queryClient';
import { router } from '../config/router';
import { auth0Config } from '../config/auth0';
import { AuthProvider } from './AuthProvider';

// Initialize i18n
import '../i18n';

export function AppProviders() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <Auth0Provider
          domain={auth0Config.domain}
          clientId={auth0Config.clientId}
          authorizationParams={{
            audience: auth0Config.audience,
            redirect_uri: auth0Config.redirectUri,
          }}
        >
          <MantineProvider theme={mantineTheme}>
            <ModalsProvider>
              <Notifications />
              <AuthProvider>
                <RouterProvider router={router} />
                <ReactQueryDevtools initialIsOpen={false} />
              </AuthProvider>
            </ModalsProvider>
          </MantineProvider>
        </Auth0Provider>
      </QueryClientProvider>
    </ErrorBoundary>
  );
}
