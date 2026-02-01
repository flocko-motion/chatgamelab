import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useRequiredAuthenticatedApi } from "../useAuthenticatedApi";
import { apiClient } from "../client";
import { queryKeys } from "../queryKeys";
import type {
  ObjAiPlatform,
  ObjSystemSettings,
  HttpxErrorResponse,
  RoutesRolesResponse,
  RoutesVersionResponse,
  RoutesUpdateSystemSettingsRequest,
} from "../generated";

// Platforms hook (requires authentication)
export function usePlatforms() {
  const api = useRequiredAuthenticatedApi();

  return useQuery<ObjAiPlatform[], HttpxErrorResponse>({
    queryKey: queryKeys.platforms,
    queryFn: () =>
      api.platforms.platformsList().then((response) => response.data),
  });
}

// Roles hook (requires authentication)
export function useRoles() {
  const api = useRequiredAuthenticatedApi();

  return useQuery<RoutesRolesResponse, HttpxErrorResponse>({
    queryKey: queryKeys.roles,
    queryFn: () => api.roles.rolesList().then((response) => response.data),
  });
}

// System Settings hook (requires authentication)
export function useSystemSettings() {
  const api = useRequiredAuthenticatedApi();

  return useQuery<ObjSystemSettings, HttpxErrorResponse>({
    queryKey: queryKeys.systemSettings,
    queryFn: () =>
      api.system.settingsList().then((response) => response.data),
  });
}

// Version hook (public endpoint, no auth needed)
export function useVersion() {
  const api = apiClient;

  return useQuery<RoutesVersionResponse, HttpxErrorResponse>({
    queryKey: queryKeys.version,
    queryFn: () => api.version.versionList().then((response) => response.data),
  });
}

// Update System Settings mutation (admin only)
export function useUpdateSystemSettings() {
  const api = useRequiredAuthenticatedApi();
  const queryClient = useQueryClient();

  return useMutation<
    ObjSystemSettings,
    HttpxErrorResponse,
    RoutesUpdateSystemSettingsRequest
  >({
    mutationFn: (request) =>
      api.system.settingsPartialUpdate(request).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.systemSettings });
    },
  });
}
