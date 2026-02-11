import { MantineProvider } from "@mantine/core";
import { ModalsProvider } from "@mantine/modals";
import { Notifications } from "@mantine/notifications";
import { QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { RouterProvider } from "@tanstack/react-router";
import { Auth0Provider } from "@auth0/auth0-react";
import { ErrorBoundary } from "@/common/components/ErrorBoundary";
import { GlobalErrorModal } from "@/common/components/GlobalErrorModal";

import { mantineTheme } from "../config/mantineTheme";
import { queryClient } from "../config/queryClient";
import { router } from "../config/router";
import { auth0Config } from "../config/auth0";
import { AuthProvider } from "./AuthProvider";
import { WorkshopModeProvider } from "./WorkshopModeProvider";

// Initialize i18n
import "../i18n";

export function AppProviders() {
  return (
    <QueryClientProvider client={queryClient}>
      <Auth0Provider
        domain={auth0Config.domain}
        clientId={auth0Config.clientId}
        authorizationParams={{
          audience: auth0Config.audience,
          redirect_uri: auth0Config.redirectUri,
        }}
        cacheLocation="localstorage"
        useRefreshTokens={true}
      >
        <MantineProvider theme={mantineTheme} forceColorScheme="light">
          <ErrorBoundary>
            <ModalsProvider>
              <Notifications />
              <GlobalErrorModal />
              <AuthProvider>
                <WorkshopModeProvider>
                  <RouterProvider router={router} />
                  <ReactQueryDevtools initialIsOpen={false} />
                </WorkshopModeProvider>
              </AuthProvider>
            </ModalsProvider>
          </ErrorBoundary>
        </MantineProvider>
      </Auth0Provider>
    </QueryClientProvider>
  );
}
