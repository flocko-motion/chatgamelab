import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { api } from "../client/api";
import { queryKeys } from "../queryKeys";
import type {
  CreateInstitutionReq,
  Error as ApiError,
  MemberSort,
  OrganisationFilter,
  OrganisationSort,
  OrganisationView,
  UpdateInstitutionReq,
  UserView,
} from "../generated/api.gen";
import { useEntityList, type List, type ListPreset } from "../lists";

/** A filter of the organisation list (the server's enum): `recentlyRegistered` keeps those registered in the last two weeks. */
export type OrganisationListFilter = OrganisationFilter;

/** An order of the organisation list (the server's enum). */
export type OrganisationListSort = OrganisationSort;

/** The organisations the caller may see, as a list. */
export function useOrganisations<F extends string, O extends OrganisationListFilter = never>(
  preset: ListPreset<F, OrganisationListSort, O>,
): List<OrganisationView, F, OrganisationListSort, O> {
  return useEntityList<OrganisationView, F, OrganisationListSort, O>(preset, {
    key: (params) => [...queryKeys.institutions, { ...params }],
    fetch: (params) =>
      api.getApiInstitutions(params as OrganisationListQuery).then((r) => r.data),
  });
}

/** An order of the member list (the server's enum). */
export type MemberListSort = MemberSort;

/** An organisation's members, as a list. */
export function useMembers<F extends string>(
  institutionId: string | undefined,
  preset: ListPreset<F, MemberListSort>,
): List<UserView, F, MemberListSort> {
  return useEntityList<UserView, F, MemberListSort>(
    preset,
    {
      key: (params) => [...queryKeys.institutionMembers(institutionId ?? ""), { ...params }],
      fetch: (params) =>
        api
          .getApiInstitutionsIdMembers(institutionId!, params as MemberListQuery)
          .then((r) => r.data),
    },
    { enabled: !!institutionId },
  );
}

// A list's parameters come from presets typed by the server's enums; the
// server refuses what it does not know.
type OrganisationListQuery = Parameters<typeof api.getApiInstitutions>[0];
type MemberListQuery = Parameters<typeof api.getApiInstitutionsIdMembers>[1];

/** One organisation as the caller sees it, with their rights on it (may* flags). */
export function useOrganisation(institutionId: string | undefined) {
  return useQuery<OrganisationView, ApiError>({
    queryKey: queryKeys.institution(institutionId ?? ""),
    queryFn: () => api.getApiInstitutionsId(institutionId!).then((r) => r.data),
    enabled: !!institutionId,
  });
}

function invalidateOrganisations(
  queryClient: ReturnType<typeof useQueryClient>,
  institutionId?: string,
) {
  queryClient.invalidateQueries({ queryKey: queryKeys.institutions });
  if (institutionId) {
    queryClient.invalidateQueries({ queryKey: queryKeys.institution(institutionId) });
  }
}

export function useCreateOrganisation() {
  const queryClient = useQueryClient();
  return useMutation<OrganisationView, ApiError, CreateInstitutionReq>({
    mutationFn: (request) => api.postApiInstitutions(request).then((r) => r.data),
    onSuccess: () => invalidateOrganisations(queryClient),
    onError: handleApiError,
  });
}

/** Changes an organisation's name or prompt constraints; a field left out stays. */
export function useUpdateOrganisation() {
  const queryClient = useQueryClient();
  return useMutation<OrganisationView, ApiError, { id: string } & UpdateInstitutionReq>({
    mutationFn: ({ id, ...changes }) => api.patchApiInstitutionsId(id, changes).then((r) => r.data),
    onSuccess: (_, { id }) => invalidateOrganisations(queryClient, id),
    onError: handleApiError,
  });
}

export function useDeleteOrganisation() {
  const queryClient = useQueryClient();
  return useMutation<unknown, ApiError, string>({
    mutationFn: (id) => api.deleteApiInstitutionsId(id).then((r) => r.data),
    onSuccess: () => invalidateOrganisations(queryClient),
    onError: handleApiError,
  });
}

/** Takes someone out of an organisation; the caller themselves, too (leaving). */
export function useRemoveMember() {
  const queryClient = useQueryClient();
  return useMutation<unknown, ApiError, { institutionId: string; userId: string }>({
    mutationFn: ({ institutionId, userId }) =>
      api.deleteApiInstitutionsIdMembersUserId(institutionId, userId).then((r) => r.data),
    onSuccess: (_, { institutionId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.institutionMembers(institutionId) });
      invalidateOrganisations(queryClient, institutionId);
      queryClient.invalidateQueries({ queryKey: queryKeys.adminUsers });
      queryClient.invalidateQueries({ queryKey: queryKeys.backendUser });
    },
    onError: handleApiError,
  });
}
