import { api } from "../client/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import type {
  KeyShareView,
  MessageResp,
  OrganisationView,
  Error as ApiError,
  ApiKeyView,
  CreateApiKeyReq,
  OwnApiKeyView,
  OwnApiKeySort,
  OrganisationKeySort,
  ShareLinkView,
} from "../generated/api.gen";
import { useEntityList, type List, type ListPreset } from "../lists";

/** The caller's keys and every share of them, for choosing a key. */
export interface OwnKeysAndShares {
  apiKeys: ApiKeyView[];
  shares: KeyShareView[];
}

/** Every one of the caller's keys, with every share of them (all pages). */
export function useApiKeys() {
  return useQuery<OwnKeysAndShares, ApiError>({
    queryKey: [...queryKeys.apiKeys, "all"],
    queryFn: () =>
      api.getApiApikeys().then(({ data }) => ({
        apiKeys: data.items.map((k) => k.key),
        shares: data.items.flatMap((k) => k.shares),
      })),
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
    queryKey: [...queryKeys.institutionApiKeys(institutionId), "all"],
    queryFn: () =>
      api.getApiInstitutionsIdApikeys(institutionId)
        .then((response) => response.data.items),
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



/** An order of the organisation's key list (the server's enum). Its filters are `platform:<id>`. */
export type OrganisationKeyListSort = OrganisationKeySort;

/** The filter that keeps the keys of one platform. */
export const platformFilter = (platformId: string) => `platform:${platformId}`;

/** The keys given to an organisation, as a list. */
export function useOrganisationApiKeys<F extends string>(
  institutionId: string | undefined,
  preset: ListPreset<F, OrganisationKeyListSort>,
): List<KeyShareView, F, OrganisationKeyListSort> {
  return useEntityList<KeyShareView, F, OrganisationKeyListSort>(
    preset,
    {
      key: (params) => [...queryKeys.institutionApiKeys(institutionId ?? ""), { ...params }],
      fetch: (params) =>
        api
          .getApiInstitutionsIdApikeys(institutionId!, params as KeyListQuery)
          .then((r) => r.data),
    },
    { enabled: !!institutionId },
  );
}

/**
 * One of the caller's own keys: the key, their own use of it (`selfShare`:
 * what makes it the default or deletes it) and every share of it.
 */
export type OwnApiKey = OwnApiKeyView;

/** An order of the own key list (the server's enum). Its filters are `platform:<id>`. */
export type OwnKeyListSort = OwnApiKeySort;

/** The caller's own keys, as a list. */
export function useOwnApiKeys<F extends string>(
  preset: ListPreset<F, OwnKeyListSort>,
): List<OwnApiKey, F, OwnKeyListSort> {
  return useEntityList<OwnApiKey, F, OwnKeyListSort>(preset, {
    key: (params) => [...queryKeys.apiKeys, { ...params }],
    fetch: (params) => api.getApiApikeys(params as OwnKeyListQuery).then((r) => r.data),
  });
}

// A list's parameters come from presets typed by the server's enums; the
// server refuses what it does not know.
type KeyListQuery = Parameters<typeof api.getApiInstitutionsIdApikeys>[1];
type OwnKeyListQuery = Parameters<typeof api.getApiApikeys>[0];
