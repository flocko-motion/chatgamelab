import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import {
  useCreateOrganisationInvite,
  useMembers,
  useRemoveMember,
  useSetUserRole,
} from "@/api/hooks";
import type { OrganisationView, UserView } from "@/api/types";
import { useAuth } from "@/providers/AuthProvider";
import { inviteErrorMessage } from "@/common/lib/invite";
import { isListNarrowed } from "@/common/lib/listNarrowed";
import { extractRawErrorCode } from "@/common/types/errorCodes";
import { MEMBER_LIST_PRESETS } from "../lib/memberListPresets";

/**
 * Everything the members tab does, for MembersTab to render: the member list,
 * what the caller sees and may do with each member, and the modals for
 * inviting a teacher, promoting, demoting, removing and leaving.
 */
export function useMembersTab(institution: OrganisationView | null | undefined) {
  const { t } = useTranslation("common");
  const navigate = useNavigate();
  const { backendUser, retryBackendFetch } = useAuth();

  const institutionId = institution?.id;
  // What the caller sees of the members and may do with them: the server's
  // views and flags, read as they are.
  const members = institution?.members ?? [];
  const canSeeEmails = members.some((m) => m.hasContact);
  const canSeeRoles = members.some((m) => !!m.role);
  const mayActOnMembers = members.some(
    (m) => m.id !== backendUser?.id && (m.mayChangeRole || m.mayRemoveMember),
  );
  const preset = canSeeRoles ? "full" : "names";
  const config = MEMBER_LIST_PRESETS[preset];

  // ── The list ────────────────────────────────────────────────────────────
  const list = useMembers(institutionId, config.list);
  const columns = config.columns.filter(
    (c) => (c !== "email" || canSeeEmails) && (c !== "actions" || mayActOnMembers),
  );

  // ── What the caller may do with a member ───────────────────────────────
  const isCurrentUser = useCallback(
    (member: UserView) => member.id === backendUser?.id,
    [backendUser?.id],
  );
  // A member's role says which change applies (staff becomes head, head
  // becomes staff); whether the caller may make it is the server's flag.
  const canPromote = useCallback(
    (member: UserView) =>
      !isCurrentUser(member) && member.mayChangeRole && member.role?.role === "staff",
    [isCurrentUser],
  );
  const canDemote = useCallback(
    (member: UserView) =>
      !isCurrentUser(member) && member.mayChangeRole && member.role?.role === "head",
    [isCurrentUser],
  );
  const canRemove = useCallback(
    (member: UserView) => !isCurrentUser(member) && member.mayRemoveMember,
    [isCurrentUser],
  );

  const setUserRole = useSetUserRole();
  const removeMember = useRemoveMember();

  // ── Inviting a teacher ──────────────────────────────────────────────────
  const createInvite = useCreateOrganisationInvite();
  const [inviteOpened, setInviteOpened] = useState(false);
  const [inviteError, setInviteError] = useState<string | null>(null);

  const inviteTeacher = async (email: string) => {
    if (!institutionId) return;
    try {
      await createInvite.mutateAsync({ institutionId, role: "staff", invitedEmail: email });
      setInviteError(null);
      setInviteOpened(false);
    } catch (error) {
      setInviteError(inviteErrorMessage(t, error));
    }
  };

  // ── Promoting, demoting, removing ───────────────────────────────────────
  const [promoting, setPromoting] = useState<UserView | null>(null);
  const [demoting, setDemoting] = useState<UserView | null>(null);
  const [removing, setRemoving] = useState<UserView | null>(null);

  const setRole = async (member: UserView | null, role: "head" | "staff", done: () => void) => {
    if (!member || !institutionId) return;
    try {
      await setUserRole.mutateAsync({ userId: member.id, role, institutionId });
      done();
    } catch {
      // shown by the mutation's error handler
    }
  };

  const confirmRemove = async () => {
    if (!removing || !institutionId) return;
    try {
      await removeMember.mutateAsync({ institutionId, userId: removing.id });
      setRemoving(null);
    } catch {
      // shown by the mutation's error handler
    }
  };

  // ── Leaving ─────────────────────────────────────────────────────────────
  const [leaveOpened, setLeaveOpened] = useState(false);
  const [leaveError, setLeaveError] = useState<string | null>(null);

  const leave = async () => {
    if (!institutionId || !backendUser?.id) return;
    try {
      await removeMember.mutateAsync({ institutionId, userId: backendUser.id });
      retryBackendFetch(); // refreshes the header
      setLeaveError(null);
      setLeaveOpened(false);
      navigate({ to: "/dashboard" });
    } catch (error) {
      setLeaveError(
        extractRawErrorCode(error) === "last_head"
          ? t("myOrganization.lastHeadError")
          : t("myOrganization.leaveError"),
      );
    }
  };

  return {
    institutionId,
    config,
    columns,
    list,
    /** Narrowed by a search, so empty means no match. */
    narrowed: isListNarrowed(list),
    mayInvite: !!institution?.mayInviteStaff,
    seesInvites: !!(institution?.mayInviteStaff || institution?.mayInviteHead),
    canSeeRoles,
    isCurrentUser,
    canPromote,
    canDemote,
    canRemove,
    requestPromote: setPromoting,
    requestDemote: setDemoting,
    requestRemove: setRemoving,
    inviteModal: {
      opened: inviteOpened,
      open: () => setInviteOpened(true),
      close: () => {
        setInviteError(null);
        setInviteOpened(false);
      },
      submit: inviteTeacher,
      loading: createInvite.isPending,
      error: inviteError,
    },
    promoteModal: {
      member: promoting,
      close: () => setPromoting(null),
      confirm: () => setRole(promoting, "head", () => setPromoting(null)),
      loading: setUserRole.isPending,
    },
    demoteModal: {
      member: demoting,
      close: () => setDemoting(null),
      confirm: () => setRole(demoting, "staff", () => setDemoting(null)),
      loading: setUserRole.isPending,
    },
    removeModal: {
      member: removing,
      close: () => setRemoving(null),
      confirm: confirmRemove,
      loading: removeMember.isPending,
    },
    leaveModal: {
      opened: leaveOpened,
      open: () => setLeaveOpened(true),
      close: () => {
        setLeaveError(null);
        setLeaveOpened(false);
      },
      confirm: leave,
      loading: removeMember.isPending,
      error: leaveError,
    },
  };
}

export type MembersTabState = ReturnType<typeof useMembersTab>;
