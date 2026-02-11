import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { useRequiredAuthenticatedApi } from "../useAuthenticatedApi";
import { queryKeys } from "../queryKeys";
import type { RoutesInviteResponse, HttpxErrorResponse } from "../generated";

// Invites hooks
export function useInvites() {
  const api = useRequiredAuthenticatedApi();

  return useQuery<RoutesInviteResponse[], HttpxErrorResponse>({
    queryKey: queryKeys.invites,
    queryFn: () => api.invites.invitesList().then((response) => response.data),
  });
}

export function useInstitutionInvites(institutionId: string | undefined) {
  const api = useRequiredAuthenticatedApi();

  return useQuery<RoutesInviteResponse[], HttpxErrorResponse>({
    queryKey: queryKeys.institutionInvites(institutionId!),
    queryFn: () => api.invites.institutionDetail(institutionId!).then((response) => response.data),
    enabled: !!institutionId,
  });
}

export function useAllInvites() {
  const api = useRequiredAuthenticatedApi();

  return useQuery<RoutesInviteResponse[], HttpxErrorResponse>({
    queryKey: queryKeys.allInvites,
    queryFn: () => api.invites.getInvites().then((response) => response.data),
  });
}

export function useRevokeInvite() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    Record<string, string>,
    HttpxErrorResponse,
    string
  >({
    mutationFn: (inviteId) =>
      api.invites.invitesDelete(inviteId).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workshops'] });
      queryClient.invalidateQueries({ queryKey: queryKeys.invites });
      queryClient.invalidateQueries({ queryKey: queryKeys.institutionInvitesBase });
      queryClient.invalidateQueries({ queryKey: queryKeys.allInvites });
    },
    onError: handleApiError,
  });
}
