import { api } from "../client/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import type { InviteResp, MessageResp, Error as ApiError } from "../generated/api.gen";

export function useInstitutionInvites(institutionId: string | undefined) {

  return useQuery<InviteResp[], ApiError>({
    queryKey: queryKeys.institutionInvites(institutionId!),
    queryFn: () => api.getApiInvitesInstitutionInstitutionId(institutionId!).then((response) => response.data),
    enabled: !!institutionId,
  });
}

export function useAllInvites() {

  return useQuery<InviteResp[], ApiError>({
    queryKey: queryKeys.allInvites,
    queryFn: () => api.getApiInvitesAll().then((response) => response.data),
  });
}

export function useRevokeInvite() {
  const queryClient = useQueryClient();

  return useMutation<
    MessageResp,
    ApiError,
    string
  >({
    mutationFn: (inviteId) =>
      api.deleteApiInvitesId(inviteId).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workshops'] });
      queryClient.invalidateQueries({ queryKey: queryKeys.invites });
      queryClient.invalidateQueries({ queryKey: queryKeys.institutionInvitesBase });
      queryClient.invalidateQueries({ queryKey: queryKeys.allInvites });
    },
    onError: handleApiError,
  });
}
