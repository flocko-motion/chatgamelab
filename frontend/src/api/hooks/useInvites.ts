import { api } from "../client/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import type {
  AcceptInviteResp,
  CreateInstitutionInviteReq,
  InviteCardView,
  InviteSort,
  MessageResp,
  Error as ApiError,
} from "../generated/api.gen";
import { useEntityList, type List, type ListPreset } from "../lists";

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

/** An order of the invite list (the server's enum). */
export type InviteListSort = InviteSort;

/**
 * The invites the caller may see, as a list: one organisation's, or with no
 * organisation every invite (admins).
 */
export function useInvites<F extends string>(
  institutionId: string | undefined,
  preset: ListPreset<F, InviteListSort>,
): List<InviteCardView, F, InviteListSort> {
  return useEntityList<InviteCardView, F, InviteListSort>(
    preset,
    institutionId
      ? {
          key: (params) => [...queryKeys.institutionInvites(institutionId), { ...params }],
          fetch: (params) =>
            api
              .getApiInvitesInstitutionInstitutionId(institutionId, params as InviteListQuery)
              .then((r) => r.data),
        }
      : {
          key: (params) => [...queryKeys.allInvites, { ...params }],
          fetch: (params) => api.getApiInvitesAll(params as InviteListQuery).then((r) => r.data),
        },
  );
}

// A list's parameters come from presets typed by the server's enums; the
// server refuses what it does not know.
type InviteListQuery = Parameters<typeof api.getApiInvitesAll>[0];

/**
 * Invites someone into an organisation (a head, a staff member). Its errors
 * are the caller's to show (`inviteErrorMessage`): a pending invite for the
 * same person answers `conflict`.
 */
export function useCreateOrganisationInvite() {
  const queryClient = useQueryClient();
  return useMutation<InviteCardView, ApiError, CreateInstitutionInviteReq>({
    mutationFn: (request) => api.postApiInvitesInstitution(request).then((r) => r.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.invites });
      queryClient.invalidateQueries({ queryKey: queryKeys.institutionInvitesBase });
      queryClient.invalidateQueries({ queryKey: queryKeys.allInvites });
    },
  });
}

/** The invites addressed to the caller; loaded only while `enabled`. */
export function useMyInvites(enabled = true) {
  return useQuery<InviteCardView[], ApiError>({
    queryKey: queryKeys.invites,
    queryFn: () => api.getApiInvites().then((r) => r.data),
    enabled,
  });
}

/**
 * Accepts one of the caller's invites, then reloads their invites and who
 * they are. `onSuccess` is the caller's follow-up; errors are the caller's.
 */
export function useAcceptInvite(options: { onSuccess?: (data: AcceptInviteResp) => void } = {}) {
  const queryClient = useQueryClient();
  return useMutation<AcceptInviteResp, ApiError, string>({
    mutationFn: (inviteId) => api.postApiInvitesIdOrTokenAccept(inviteId).then((r) => r.data),
    onSuccess: (data) => {
      queryClient.refetchQueries({ queryKey: queryKeys.invites });
      queryClient.refetchQueries({ queryKey: queryKeys.currentUser });
      options.onSuccess?.(data);
    },
  });
}

/**
 * Declines one of the caller's invites, then reloads their invites.
 * `onSuccess` is the caller's follow-up; errors are the caller's.
 */
export function useDeclineInvite(options: { onSuccess?: (data: MessageResp) => void } = {}) {
  const queryClient = useQueryClient();
  return useMutation<MessageResp, ApiError, string>({
    mutationFn: (inviteId) => api.postApiInvitesIdDecline(inviteId).then((r) => r.data),
    onSuccess: (data) => {
      queryClient.refetchQueries({ queryKey: queryKeys.invites });
      options.onSuccess?.(data);
    },
  });
}

/**
 * One invite by its id or link token, read imperatively by the invite link
 * page, which handles every failure itself (no cache, no retry).
 */
export const fetchInvite = (idOrToken: string): Promise<InviteCardView> =>
  api.getApiInvitesIdOrToken(idOrToken).then((r) => r.data);

/**
 * Accepts an invite by its id or link token, imperatively, for the invite
 * link page, which handles every failure itself and reloads the app after.
 */
export const acceptInvite = (idOrToken: string): Promise<AcceptInviteResp> =>
  api.postApiInvitesIdOrTokenAccept(idOrToken).then((r) => r.data);
