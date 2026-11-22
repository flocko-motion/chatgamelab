import { api } from "../client/api";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import type {
  GenreInfo,
  PlatformInfo,
  ThemeInfo,
  SystemSettings,
  Error as ApiError,
  GetVersionResp,
  UpdateSystemSettingsReq,
  SystemConstraintsResp,
} from "../generated/api.gen";

// Platforms hook (requires authentication)
export function usePlatforms() {

  return useQuery<PlatformInfo[], ApiError>({
    queryKey: queryKeys.platforms,
    queryFn: () =>
      api.getApiPlatforms().then((response) => response.data),
  });
}


// The engine's genres, each with the features it needs, for the game editor
export function useGenres() {

  return useQuery<GenreInfo[], ApiError>({
    queryKey: queryKeys.genres,
    queryFn: () => api.getApiGenres().then((response) => response.data),
    staleTime: Infinity,
  });
}

// The playback-engine's looks, the default first, for the game editor
export function useThemes() {

  return useQuery<ThemeInfo[], ApiError>({
    queryKey: queryKeys.themes,
    queryFn: () => api.getApiThemes().then((response) => response.data),
    staleTime: Infinity,
  });
}

// System Settings hook (requires authentication)
export function useSystemSettings() {

  return useQuery<SystemSettings, ApiError>({
    queryKey: queryKeys.systemSettings,
    queryFn: () => api.getApiSystemSettings().then((response) => response.data),
  });
}

// Version hook (public endpoint, no auth needed)
export function useVersion() {

  return useQuery<GetVersionResp, ApiError>({
    queryKey: queryKeys.version,
    queryFn: () => api.getApiVersion().then((response) => response.data),
  });
}

// Update System Settings mutation (admin only)
export function useUpdateSystemSettings() {
  const queryClient = useQueryClient();

  return useMutation<
    SystemSettings,
    ApiError,
    UpdateSystemSettingsReq
  >({
    mutationFn: (request) =>
      api.patchApiSystemSettings(request)
        .then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.systemSettings });
    },
  });
}

// Set/clear system free-use API key (admin only)
export function useSetSystemFreeUseKey() {
  const queryClient = useQueryClient();

  return useMutation<
    SystemSettings,
    ApiError,
    { apiKeyId: string | null }
  >({
    mutationFn: ({ apiKeyId }) =>
      api.patchApiSystemSettingsFreeUseKey({
          apiKeyId: apiKeyId ?? undefined,
        })
        .then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.systemSettings });
    },
    onError: handleApiError,
  });
}

/** The platform's own prompt constraints; loaded only while `enabled`. */
export function useSystemConstraints(enabled = true) {
  return useQuery<SystemConstraintsResp, ApiError>({
    queryKey: queryKeys.systemConstraints,
    queryFn: () => api.getApiSystemConstraints().then((r) => r.data),
    enabled,
  });
}
