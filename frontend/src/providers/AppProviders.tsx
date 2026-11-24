import type { ReactNode } from "react";
import { MantineProvider } from "@mantine/core";
import { ModalsProvider } from "@mantine/modals";
import { Notifications } from "@mantine/notifications";
import { QueryClientProvider } from "@tanstack/react-query";
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
import { SiteProvider } from "./SiteProvider";

// Initialize i18n
import "../i18n";

/** Auth0's SDK, when the instance offers Auth0 sign-in. */
function SignInProvider({ children }: { children: ReactNode }) {
  if (!auth0Config) return <>{children}</>;
  return (
    <Auth0Provider
      domain={auth0Config.domain}
      clientId={auth0Config.clientId}
      authorizationParams={{
        redirect_uri: auth0Config.redirectUri,
        scope: "openid profile offline_access",
      }}
      cacheLocation="localstorage"
      useRefreshTokens={true}
    >
      {children}
    </Auth0Provider>
  );
}

export function AppProviders() {
  return (
    <QueryClientProvider client={queryClient}>
      <SignInProvider>
        <MantineProvider theme={mantineTheme} forceColorScheme="light">
          <ErrorBoundary>
            <ModalsProvider>
              <Notifications />
              <GlobalErrorModal />
              <SiteProvider>
                <AuthProvider>
                  <WorkshopModeProvider>
                    <RouterProvider router={router} />
                  </WorkshopModeProvider>
                </AuthProvider>
              </SiteProvider>
            </ModalsProvider>
          </ErrorBoundary>
        </MantineProvider>
      </SignInProvider>
    </QueryClientProvider>
  );
}
