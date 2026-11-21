import { api } from "../client/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import type {
  KeyShareView,
  MessageResp,
  OrganisationView,
  Error as ApiError,
  ApiKeysResp,
  CreateApiKeyReq,
  ShareLinkView,
} from "../generated/api.gen";

// API Keys hooks
export function useApiKeys() {

  return useQuery<ApiKeysResp, ApiError>({
    queryKey: queryKeys.apiKeys,
    queryFn: () => api.getApiApikeys().then((response) => response.data),
  });
}

export function useCreateApiKey() {
  const queryClient = useQueryClient();

  return useMutation<
    KeyShareView,
    ApiError,
    CreateApiKeyReq
  >({
    mutationFn: (request) =>
      api.postApiApikeysNew(request).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}


export function useDeleteApiKey() {
  const queryClient = useQueryClient();

  return useMutation<
    MessageResp,
    ApiError,
    { id: string; cascade?: boolean }
  >({
    mutationFn: ({ id, cascade }) =>
      api.deleteApiApikeysId(id, { cascade: cascade === undefined ? undefined : String(cascade) })
        .then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
      queryClient.invalidateQueries({ queryKey: queryKeys.workshops });
      queryClient.invalidateQueries({ queryKey: queryKeys.institutions });
      queryClient.invalidateQueries({ queryKey: ["institutionApiKeys"] });
    },
    onError: (error) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
      handleApiError(error);
    },
  });
}

export function useSetDefaultApiKey() {
  const queryClient = useQueryClient();

  return useMutation<KeyShareView, ApiError, { id: string }>({
    mutationFn: ({ id }) =>
      api.putApiApikeysIdDefault(id).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

// Institution API Keys hooks
export function useInstitutionApiKeys(institutionId: string) {

  return useQuery<KeyShareView[], ApiError>({
    queryKey: queryKeys.institutionApiKeys(institutionId),
    queryFn: () =>
      api.getApiInstitutionsIdApikeys(institutionId)
        .then((response) => response.data),
    enabled: !!institutionId,
  });
}

// Game shares for a specific API key share
// context: "personal" = all shares (owner only), "organization" = organisation shares only
export function useApiKeyGameShares(
  shareId: string | null,
  context?: "personal" | "organization",
) {

  return useQuery<ShareLinkView[], ApiError>({
    queryKey: [...queryKeys.apiKeyGameShares(shareId ?? ""), context ?? "all"],
    queryFn: () =>
      api.getApiApikeysIdGameShares(shareId!, { context })
        .then((response) => response.data),
    enabled: !!shareId,
  });
}

export function useShareApiKeyWithInstitution() {
  const queryClient = useQueryClient();

  return useMutation<
    KeyShareView,
    ApiError,
    {
      shareId: string;
      institutionId: string;
    }
  >({
    mutationFn: ({ shareId, institutionId }) =>
      api.postApiApikeysIdShares(shareId, {
          institutionId,
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

  return useMutation<
    MessageResp,
    ApiError,
    { shareId: string; institutionId: string }
  >({
    mutationFn: ({ shareId }) =>
      api.deleteApiApikeysId(shareId, { cascade: "false" })
        .then((response) => response.data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
      queryClient.invalidateQueries({
        queryKey: queryKeys.institutionApiKeys(variables.institutionId),
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.workshops });
      queryClient.invalidateQueries({
        queryKey: queryKeys.institution(variables.institutionId),
      });
    },
    onError: handleApiError,
  });
}

// Institution Free-Use Key hook
export function useSetInstitutionFreeUseKey() {
  const queryClient = useQueryClient();

  return useMutation<
    OrganisationView,
    ApiError,
    { institutionId: string; shareId: string | null }
  >({
    mutationFn: ({ institutionId, shareId }) =>
      api.patchApiInstitutionsIdFreeUseKey(institutionId, {
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


