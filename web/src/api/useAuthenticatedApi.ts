import { useMemo } from "react";
import { useAuth } from "@/providers/AuthProvider";
import { Api } from "./generated";
import { createAuthenticatedApiConfig } from "./client/http";

/**
 * Hook that provides an authenticated API client.
 * The client automatically injects the Bearer token on all requests.
 *
 * @throws Error if user is not authenticated and a request is made
 */
export function useAuthenticatedApi() {
  const { getAccessToken, isAuthenticated } = useAuth();

  const api = useMemo(() => {
    if (!isAuthenticated) {
      return null;
    }
    return new Api(createAuthenticatedApiConfig(getAccessToken));
  }, [getAccessToken, isAuthenticated]);

  return api;
}

/**
 * Hook that provides an authenticated API client, throwing if not authenticated.
 * Use this in components that are guaranteed to be rendered only when authenticated.
 *
 * IMPORTANT: Instead of throwing synchronously during render (which causes
 * ErrorBoundary → re-render → throw → infinite loop), this returns a Proxy
 * that defers the error to when an API method is actually called. This prevents
 * browser freezes when auth state briefly becomes false (e.g. token refresh).
 */
export function useRequiredAuthenticatedApi(): Api<unknown> {
  const api = useAuthenticatedApi();

  return useMemo(() => {
    if (api) {
      return api;
    }

    // Return a Proxy that throws only when a property is accessed and then called.
    // This allows the component to render without triggering an ErrorBoundary loop.
    // TanStack Query hooks will catch the error in queryFn / mutationFn instead.
    const handler: ProxyHandler<object> = {
      get(_target, prop) {
        // Allow typeof checks and symbol access (used by React internals)
        if (typeof prop === "symbol" || prop === "then" || prop === "toJSON") {
          return undefined;
        }
        // Return a nested proxy for namespace access (e.g. api.games.gamesList)
        return new Proxy(() => {}, {
          get(_t, innerProp) {
            if (typeof innerProp === "symbol") return undefined;
            return () => {
              throw new Error(
                `API call ${String(prop)}.${String(innerProp)}() failed: user is not authenticated`,
              );
            };
          },
          apply() {
            throw new Error(
              `API call ${String(prop)}() failed: user is not authenticated`,
            );
          },
        });
      },
    };
    return new Proxy({}, handler) as Api<unknown>;
  }, [api]);
}
