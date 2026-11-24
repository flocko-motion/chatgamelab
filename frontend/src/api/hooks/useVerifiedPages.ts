import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { api } from "../client/api";
import { queryKeys } from "../queryKeys";
import type {
  CreateVerifiedPageReq,
  Error as ApiError,
  MessageResp,
  VerifiedPageView,
} from "../generated/api.gen";

/**
 * An organisation's verified pages: the pages on its own website that may
 * frame its embedded games after the host's yes. The server answers them to
 * the organisation's head and staff; anyone else gets an error, and no list.
 */
export function useVerifiedPages(institutionId: string | undefined) {
  return useQuery<VerifiedPageView[], ApiError>({
    queryKey: queryKeys.verifiedPages(institutionId ?? ""),
    queryFn: () =>
      api.getApiInstitutionsIdVerifiedPages(institutionId!).then((r) => r.data.pages),
    enabled: !!institutionId,
    // A refusal means the caller may not see them: asking again changes nothing.
    retry: false,
  });
}

/** Registers a verified page for an organisation; the server makes its page ID. */
export function useCreateVerifiedPage() {
  const queryClient = useQueryClient();
  return useMutation<VerifiedPageView, ApiError, { institutionId: string } & CreateVerifiedPageReq>({
    mutationFn: ({ institutionId, ...page }) =>
      api.postApiInstitutionsIdVerifiedPages(institutionId, page).then((r) => r.data),
    onSuccess: (_, { institutionId }) =>
      queryClient.invalidateQueries({ queryKey: queryKeys.verifiedPages(institutionId) }),
    onError: handleApiError,
  });
}

/** Removes a verified page; the games embedded for it start nothing until they get another. */
export function useDeleteVerifiedPage() {
  const queryClient = useQueryClient();
  return useMutation<MessageResp, ApiError, { id: string; institutionId: string }>({
    mutationFn: ({ id }) => api.deleteApiVerifiedPagesId(id).then((r) => r.data),
    onSuccess: (_, { institutionId }) =>
      queryClient.invalidateQueries({ queryKey: queryKeys.verifiedPages(institutionId) }),
    onError: handleApiError,
  });
}
