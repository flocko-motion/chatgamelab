import { useEffect, useRef, useState } from "react";
import { useDisclosure } from "@mantine/hooks";
import { useNavigate } from "@tanstack/react-router";
import {
  useAcceptInvite,
  useAcknowledgeAllNotifications,
  useAcknowledgeNotification,
  useDeclineInvite,
  useMyInvites,
  useNotifications,
} from "@/api/hooks";
import type { InviteCardView, NotificationType, NotificationView } from "@/api/types";
import { ADMIN_VERIFIED_PAGES_ANCHOR, ROUTES } from "@/common/routes/routes";
import { useAuth } from "@/providers/AuthProvider";

const SEEN_INVITES_KEY = "cgl_seen_invites";

function seenInviteIds(): Set<string> {
  try {
    const stored = localStorage.getItem(SEEN_INVITES_KEY);
    return new Set(stored ? (JSON.parse(stored) as string[]) : []);
  } catch {
    return new Set();
  }
}

function markInvitesSeen(ids: string[]): void {
  try {
    const seen = seenInviteIds();
    ids.forEach((id) => seen.add(id));
    localStorage.setItem(SEEN_INVITES_KEY, JSON.stringify([...seen]));
  } catch {
    // storage unavailable: the panel opens again next time
  }
}

/** Leads to where the caller acts on an entry of its type. */
function navigateToTarget(navigate: ReturnType<typeof useNavigate>, n: NotificationView) {
  const ownVerifiedPage = () => navigate({ to: ROUTES.MY_ORGANIZATION, search: { tab: "pages" } });
  const targets: Record<NotificationType, () => unknown> = {
    newOrganisations: () => navigate({ to: ROUTES.ADMIN_ORGANIZATIONS }),
    pagesWaiting: () => navigate({ to: ROUTES.ADMIN_ORGANIZATIONS, hash: ADMIN_VERIFIED_PAGES_ANCHOR }),
    pageApproved: ownVerifiedPage,
    pageTurnedDown: ownVerifiedPage,
  };
  targets[n.type]();
}

/** What an invite leads into: the workshop (with its organisation) or the organisation. */
function inviteTarget(invite: InviteCardView): string {
  if (!invite.workshopId) return invite.institutionName || invite.institutionId || "-";
  const workshop = invite.workshopName || invite.workshopId;
  return invite.institutionName ? `${workshop} (${invite.institutionName})` : workshop;
}

/** A pending invite as the bell shows it. */
export interface BellInvite {
  invite: InviteCardView;
  /** An invite into a workshop; otherwise into an organisation. */
  workshop: boolean;
  target: string;
}

/**
 * Counts the times `count` went up, from nothing on the first load: the bell
 * wiggles once for each.
 */
function useRises(count: number, ready: boolean): number {
  const [state, setState] = useState<{ count: number | null; rises: number }>({ count: null, rises: 0 });
  if (ready && count !== state.count) {
    setState({ count, rises: state.rises + (count > (state.count ?? 0) ? 1 : 0) });
  }
  return state.rises;
}

/**
 * The header's bell: the caller's notifications and their pending invites,
 * as entries of one panel. Clicking a notification leads to its target and
 * acknowledges it; invites are accepted or declined. The bell wiggles once
 * each time the number of entries goes up, and the panel opens by itself
 * once for invites the caller has not seen.
 */
export function useNotificationBell() {
  const navigate = useNavigate();
  const { retryBackendFetch } = useAuth();
  const [opened, { open, close, toggle }] = useDisclosure(false);

  const { data: notifications = [], isPending: notificationsPending } = useNotifications();
  const { data: invites, isPending: invitesPending, isSuccess: invitesLoaded } = useMyInvites();
  const pendingInvites = (invites ?? []).filter((inv) => inv.status === "pending");
  const acknowledge = useAcknowledgeNotification();
  const acknowledgeAllMutation = useAcknowledgeAllNotifications();

  const count = notifications.length + pendingInvites.length;
  // a failed query counts as answered: the bell still wiggles for the other
  const wiggles = useRises(count, !notificationsPending && !invitesPending);

  const markPendingSeen = () => markInvitesSeen(pendingInvites.map((inv) => inv.id));

  // Open the panel once for invites the caller has not seen.
  const autoOpened = useRef(false);
  const hasUnseenInvites = pendingInvites.some((inv) => !seenInviteIds().has(inv.id));
  useEffect(() => {
    if (invitesLoaded && hasUnseenInvites && !autoOpened.current) {
      autoOpened.current = true;
      open();
    }
  }, [invitesLoaded, hasUnseenInvites, open]);

  const closePanel = () => {
    markPendingSeen();
    close();
  };

  /** Leads to the entry's target and acknowledges it up to what the caller saw. */
  const openNotification = (n: NotificationView) => {
    acknowledge.mutate({ type: n.type, subject: n.subject, at: n.at });
    closePanel();
    navigateToTarget(navigate, n);
  };

  const acknowledgeAll = () => acknowledgeAllMutation.mutate();

  const [accepting, setAccepting] = useState<InviteCardView | null>(null);
  const acceptMutation = useAcceptInvite({
    onSuccess: () => {
      markPendingSeen();
      retryBackendFetch(); // the caller's organisation changed
      close();
      const workshopInvite = !!accepting?.workshopId;
      setAccepting(null);
      setTimeout(() => navigate({ to: workshopInvite ? (ROUTES.MY_WORKSHOP as "/") : ROUTES.MY_ORGANIZATION }), 100);
    },
  });
  const acceptInvite = (invite: InviteCardView) => {
    setAccepting(invite);
    acceptMutation.mutate(invite.id);
  };

  const [declining, setDeclining] = useState<InviteCardView | null>(null);
  const declineMutation = useDeclineInvite({ onSuccess: () => setDeclining(null) });
  const confirmDecline = () => {
    if (declining) declineMutation.mutate(declining.id);
  };

  return {
    /** The panel is open. */
    opened,
    toggle,
    close: closePanel,
    /** The entries the bell counts: notifications and pending invites. */
    count,
    /** Bumped each time the count goes up: the bell wiggles once per bump. */
    wiggles,
    notifications,
    openNotification,
    acknowledgeAll,
    acknowledgingAll: acknowledgeAllMutation.isPending,
    invites: pendingInvites.map((invite): BellInvite => ({
      invite,
      workshop: !!invite.workshopId,
      target: inviteTarget(invite),
    })),
    acceptInvite,
    accepting: acceptMutation.isPending,
    /** The invite whose declining waits for confirmation. */
    declining,
    askDecline: setDeclining,
    cancelDecline: () => setDeclining(null),
    confirmDecline,
    decliningPending: declineMutation.isPending,
  };
}
