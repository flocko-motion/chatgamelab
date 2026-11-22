/**
 * Organisations registering themselves: the declaration, a head's consent and
 * its withdrawal, admins' suspensions and the organisations registered lately.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../client/api";
import { queryKeys } from "../queryKeys";
import type {
  ConsentToDeclarationReq,
  DeclarationResp,
  Error as ApiError,
  OrganisationView,
  OrganisationConsentView,
  OrganisationSuspensionView,
} from "../generated/api.gen";

/** The declaration organisations consent to, with the restrictions suggested beside it. */
export function useDeclaration() {
  return useQuery<DeclarationResp, ApiError>({
    queryKey: queryKeys.declaration,
    queryFn: () => api.getApiLegalInstitutionDeclaration().then((r) => r.data),
    staleTime: Infinity,
  });
}

/** An organisation as its heads consent for it: its consent status and details. */
export function useOrganisationConsent(institutionId: string | undefined) {
  return useQuery<OrganisationConsentView, ApiError>({
    queryKey: queryKeys.organisationConsent(institutionId ?? ""),
    queryFn: () => api.getApiInstitutionsIdDeclarationConsent(institutionId!).then((r) => r.data),
    enabled: !!institutionId,
  });
}

/** A head's consent to the current declaration, for their organisation. */
export function useConsentToDeclaration() {
  const queryClient = useQueryClient();
  return useMutation<
    OrganisationConsentView,
    ApiError,
    { institutionId: string; consent: Omit<ConsentToDeclarationReq, "id"> }
  >({
    mutationFn: ({ institutionId, consent }) =>
      api.postApiInstitutionsIdDeclarationConsent(institutionId, consent).then((r) => r.data),
    onSuccess: (data, { institutionId }) => {
      queryClient.setQueryData(queryKeys.organisationConsent(institutionId), data);
      queryClient.invalidateQueries({ queryKey: queryKeys.institution(institutionId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.institutions });
    },
  });
}

/** A head withdraws their organisation's consent, which suspends it. */
export function useWithdrawConsent() {
  const queryClient = useQueryClient();
  return useMutation<OrganisationConsentView, ApiError, string>({
    mutationFn: (institutionId) =>
      api.deleteApiInstitutionsIdDeclarationConsent(institutionId).then((r) => r.data),
    onSuccess: (data, institutionId) => {
      queryClient.setQueryData(queryKeys.organisationConsent(institutionId), data);
    },
  });
}

/** An admin suspends an organisation (with a reason) or lifts that (reason null). */
export function useSetOrganisationSuspension() {
  const queryClient = useQueryClient();
  return useMutation<OrganisationSuspensionView, ApiError, { institutionId: string; reason: string | null }>({
    mutationFn: ({ institutionId, reason }) =>
      (reason === null
        ? api.deleteApiInstitutionsIdSuspension(institutionId)
        : api.postApiInstitutionsIdSuspension(institutionId, { reason })
      ).then((r) => r.data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.institutions }),
  });
}

/** How often the organisations registered lately are checked again. */
const RECENT_REFRESH_MS = 2 * 60 * 1000;

/**
 * The organisations that registered themselves in the last two weeks, which
 * admins check, checked again every two minutes.
 */
export function useRecentlyRegisteredInstitutions(options: { enabled?: boolean } = {}) {
  return useQuery<OrganisationView[], ApiError>({
    queryKey: queryKeys.recentlyRegisteredInstitutions,
    queryFn: () =>
      api.getApiInstitutions({ filter: ["recentlyRegistered"] }).then((r) => r.data.items),
    refetchInterval: RECENT_REFRESH_MS,
    enabled: options.enabled ?? true,
  });
}
