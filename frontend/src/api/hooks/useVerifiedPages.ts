import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { api } from "../client/api";
import { queryKeys } from "../queryKeys";
import type {
  CreateVerifiedPageReq,
  Error as ApiError,
  MessageResp,
  VerifiedPageFilter,
  VerifiedPageSort,
  UpdateVerifiedPageReq,
  VerifiedPageView,
} from "../generated/api.gen";
import { useEntityList, type List, type ListPreset } from "../lists";
import { ErrorCodes, extractRawErrorCode } from "@/common/types/errorCodes";

/** A form field of a verified page: its address or its callback. */
export type VerifiedPageField = "pageUrl" | "callbackUrl";

/** The field each refusal of a page's addresses is about; the form shows it there. */
const FIELD_OF_CODE: Record<string, VerifiedPageField> = {
  [ErrorCodes.PAGE_URL_INVALID]: "pageUrl",
  [ErrorCodes.PAGE_URL_NOT_HTTPS]: "pageUrl",
  [ErrorCodes.CALLBACK_URL_INVALID]: "callbackUrl",
  [ErrorCodes.CALLBACK_URL_NOT_HTTPS]: "callbackUrl",
  [ErrorCodes.CALLBACK_OFF_SITE]: "callbackUrl",
};

/** The field and code of a refusal about one of a page's addresses; null for any other error. */
export function verifiedPageFieldError(error: unknown): { field: VerifiedPageField; code: string } | null {
  const code = extractRawErrorCode(error);
  const field = code ? FIELD_OF_CODE[code] : undefined;
  return code && field ? { field, code } : null;
}

/** A refusal about an address shows at its field; any other error as usual. */
function handleSaveError(error: ApiError) {
  if (!verifiedPageFieldError(error)) handleApiError(error);
}

/** A filter of the verified page list (the server's enum): waiting or approved. */
export type VerifiedPageListFilter = VerifiedPageFilter;

/** An order of the verified page list (the server's enum). */
export type VerifiedPageListSort = VerifiedPageSort;

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
    onError: handleSaveError,
  });
}

/** Changes a verified page's address and callback; its page ID stays, a change waits for approval again. */
export function useUpdateVerifiedPage() {
  const queryClient = useQueryClient();
  return useMutation<VerifiedPageView, ApiError, { id: string; institutionId: string } & UpdateVerifiedPageReq>({
    mutationFn: ({ id, pageUrl, callbackUrl }) =>
      api.patchApiVerifiedPagesId(id, { pageUrl, callbackUrl }).then((r) => r.data),
    onSuccess: (_, { institutionId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.verifiedPages(institutionId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.allVerifiedPages });
    },
    onError: handleSaveError,
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

/** The verified pages the caller may see, waiting first, as a list: the admins' allow list. */
export function useAllVerifiedPages<F extends string, O extends VerifiedPageListFilter = never>(
  preset: ListPreset<F, VerifiedPageListSort, O>,
): List<VerifiedPageView, F, VerifiedPageListSort, O> {
  return useEntityList<VerifiedPageView, F, VerifiedPageListSort, O>(preset, {
    key: (params) => [...queryKeys.allVerifiedPages, { ...params }],
    fetch: (params) =>
      api.getApiVerifiedPages(params as Parameters<typeof api.getApiVerifiedPages>[0]).then((r) => r.data),
  });
}

/** Approves a verified page, or withdraws the approval; for an admin. */
export function useApproveVerifiedPage() {
  const queryClient = useQueryClient();
  return useMutation<VerifiedPageView, ApiError, { id: string; approved: boolean }>({
    mutationFn: ({ id, approved }) =>
      api.putApiVerifiedPagesIdApproval(id, { approved }).then((r) => r.data),
    onSuccess: (page) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.allVerifiedPages });
      queryClient.invalidateQueries({ queryKey: queryKeys.verifiedPages(page.organisationId) });
    },
    onError: handleApiError,
  });
}
