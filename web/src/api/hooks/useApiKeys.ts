import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { useRequiredAuthenticatedApi } from "../useAuthenticatedApi";
import { queryKeys } from "../queryKeys";
import type {
  ObjApiKeyShare,
  ObjInstitution,
  HttpxErrorResponse,
  RoutesCreateApiKeyRequest,
  RoutesShareRequest,
} from "../generated";

// API Keys hooks
export function useApiKeys() {
  const api = useRequiredAuthenticatedApi();

  return useQuery<ObjApiKeyShare[], HttpxErrorResponse>({
    queryKey: queryKeys.apiKeys,
    queryFn: () => api.apikeys.apikeysList().then((response) => response.data),
  });
}

export function useCreateApiKey() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    ObjApiKeyShare,
    HttpxErrorResponse,
    RoutesCreateApiKeyRequest
  >({
    mutationFn: (request) =>
      api.apikeys.postApikeys(request).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

export function useShareApiKey() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    ObjApiKeyShare,
    HttpxErrorResponse,
    { id: string; request: RoutesShareRequest }
  >({
    mutationFn: ({ id, request }) =>
      api.apikeys.sharesCreate(id, request).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

export function useUpdateApiKeyName() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    ObjApiKeyShare,
    HttpxErrorResponse,
    { id: string; name: string }
  >({
    mutationFn: ({ id, name }) =>
      api.apikeys
        .apikeysPartialUpdate(id, { name })
        .then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

export function useDeleteApiKey() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    ObjApiKeyShare,
    HttpxErrorResponse,
    { id: string; cascade?: boolean }
  >({
    mutationFn: ({ id, cascade }) =>
      api.apikeys
        .apikeysDelete(id, { cascade })
        .then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

export function useSetDefaultApiKey() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<ObjApiKeyShare, HttpxErrorResponse, { id: string }>({
    mutationFn: ({ id }) =>
      api.apikeys.defaultUpdate(id).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

// Institution API Keys hooks
export function useInstitutionApiKeys(institutionId: string) {
  const api = useRequiredAuthenticatedApi();

  return useQuery<ObjApiKeyShare[], HttpxErrorResponse>({
    queryKey: queryKeys.institutionApiKeys(institutionId),
    queryFn: () =>
      api.institutions
        .apikeysList(institutionId)
        .then((response) => response.data),
    enabled: !!institutionId,
  });
}

export function useShareApiKeyWithInstitution() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    ObjApiKeyShare,
    HttpxErrorResponse,
    {
      shareId: string;
      institutionId: string;
      allowPublicGameSponsoring?: boolean;
    }
  >({
    mutationFn: ({ shareId, institutionId, allowPublicGameSponsoring }) =>
      api.apikeys
        .sharesCreate(shareId, {
          institutionId,
          allowPublicGameSponsoring: allowPublicGameSponsoring ?? false,
        })
        .then((response) => response.data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionApiKeys(variables.institutionId),
      });
    },
    onError: handleApiError,
  });
}

export function useRemoveInstitutionApiKeyShare() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    ObjApiKeyShare,
    HttpxErrorResponse,
    { shareId: string; institutionId: string }
  >({
    mutationFn: ({ shareId }) =>
      api.apikeys
        .apikeysDelete(shareId, { cascade: false })
        .then((response) => response.data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionApiKeys(variables.institutionId),
      });
    },
    onError: handleApiError,
  });
}

// Institution Free-Use Key hook
export function useSetInstitutionFreeUseKey() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    ObjInstitution,
    HttpxErrorResponse,
    { institutionId: string; shareId: string | null }
  >({
    mutationFn: ({ institutionId, shareId }) =>
      api.institutions
        .freeUseKeyPartialUpdate(institutionId, {
          shareId: shareId ?? undefined,
        })
        .then((response) => response.data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.institution(variables.institutionId),
      });
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionApiKeys(variables.institutionId),
      });
    },
    onError: handleApiError,
  });
}

// Available Keys for Game hook
export function useAvailableKeysForGame(gameId: string | undefined) {
  const api = useRequiredAuthenticatedApi();

  return useQuery({
    queryKey: queryKeys.availableKeys(gameId!),
    queryFn: () =>
      api.games.availableKeysList(gameId!).then((response) => response.data),
    enabled: !!gameId,
  });
}
