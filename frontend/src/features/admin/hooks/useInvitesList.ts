import { useCallback } from "react";
import { useInvites, useRevokeInvite } from "@/api/hooks";
import type { InviteCardView } from "@/api/types";
import { INVITE_LIST_PRESETS, type InviteListPreset } from "../lib/inviteListPresets";

/**
 * An invite list, for InvitesList to render: one organisation's invites, or
 * with no organisation every invite (admins), and revoking one.
 */
export function useInvitesList(institutionId: string | undefined, preset: InviteListPreset) {
  const list = useInvites(institutionId, INVITE_LIST_PRESETS[preset].list);
  const revokeInvite = useRevokeInvite();

  /** A pending invite the caller may take back. */
  const canRevoke = useCallback(
    (invite: InviteCardView) => invite.status === "pending" && invite.mayRevoke,
    [],
  );

  return {
    config: INVITE_LIST_PRESETS[preset],
    list,
    canRevoke,
    revoke: (invite: InviteCardView) => revokeInvite.mutate(invite.id),
    revoking: revokeInvite.isPending,
  };
}
