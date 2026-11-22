import { useEffect, useRef, useCallback, useMemo, useState } from "react";
import {
  ActionIcon,
  Indicator,
  Modal,
  Stack,
  Text,
  Card,
  Group,
  Badge,
  Button,
  Loader,
  Center,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { IconBell } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import { useAuth } from "@/providers/AuthProvider";
import { useAcceptInvite, useDeclineInvite, useMyInvites } from "@/api/hooks";
import type { InviteCardView } from "@/api/types";
import { useTranslateRole } from "@/common/lib/roles";
import { ROUTES } from "@/common/routes/routes";
import classes from "./NotificationBell.module.css";
import { AdminRegistrationBell } from "./AdminRegistrationBell";

const SEEN_INVITES_KEY = "cgl_seen_invites";

function getSeenInviteIds(): Set<string> {
  try {
    const stored = localStorage.getItem(SEEN_INVITES_KEY);
    if (!stored) return new Set();
    return new Set(JSON.parse(stored));
  } catch {
    return new Set();
  }
}

function markInvitesAsSeen(inviteIds: string[]): void {
  try {
    const seen = getSeenInviteIds();
    inviteIds.forEach((id) => seen.add(id));
    localStorage.setItem(SEEN_INVITES_KEY, JSON.stringify([...seen]));
  } catch {
    // Ignore storage errors
  }
}

export function NotificationBell() {
  const { t } = useTranslation("common");
  const { backendUser, retryBackendFetch, standing } = useAuth();
  const [opened, { open, close }] = useDisclosure(false);
  const [
    declineModalOpened,
    { open: openDeclineModal, close: closeDeclineModal },
  ] = useDisclosure(false);
  const [inviteToDecline, setInviteToDecline] =
    useState<InviteCardView | null>(null);
  const hasAutoOpened = useRef(false);

  // Admins don't have invites - disable query and hide the bell
  const isAdmin = backendUser?.role?.role === "admin";

  const { data: invites, isLoading } = useMyInvites(!isAdmin); // Don't fetch invites for admins

  const navigate = useNavigate();

  const [acceptingInvite, setAcceptingInvite] =
    useState<InviteCardView | null>(null);

  const acceptMutation = useAcceptInvite({
    onSuccess: () => {
      // Mark all invites as seen to prevent auto-reopen
      const inviteIds = pendingInvites
        .map((inv) => inv.id)
        .filter(Boolean) as string[];
      if (inviteIds.length > 0) {
        markInvitesAsSeen(inviteIds);
      }

      retryBackendFetch(); // Refresh user's organization data
      close(); // Close the notifications modal

      const isWorkshopInvite = !!acceptingInvite?.workshopId;
      setAcceptingInvite(null);

      // Navigate to the appropriate page
      setTimeout(() => {
        if (isWorkshopInvite) {
          navigate({ to: ROUTES.MY_WORKSHOP as "/" });
        } else {
          navigate({ to: "/my-organization" });
        }
      }, 100);
    },
  });

  const declineMutation = useDeclineInvite({
    onSuccess: () => {
      closeDeclineModal();
      setInviteToDecline(null);
      // If no more pending invites, close the main modal
      if (pendingInvites.length <= 1) {
        handleClose();
      }
    },
  });

  /* eslint-disable react-hooks/preserve-manual-memoization -- React Compiler limitation */
  const pendingInvites = useMemo(
    () => invites?.filter((inv) => inv.status === "pending") || [],
    [invites],
  );
  /* eslint-enable react-hooks/preserve-manual-memoization */
  const pendingCount = pendingInvites.length;

  // Check for unseen invites
  const seenIds = getSeenInviteIds();
  const unseenInvites = pendingInvites.filter(
    (inv) => inv.id && !seenIds.has(inv.id),
  );
  const hasUnseenInvites = unseenInvites.length > 0;

  // Mark invites as seen when modal closes
  /* eslint-disable react-hooks/preserve-manual-memoization -- React Compiler limitation */
  const handleClose = useCallback(() => {
    const inviteIds = pendingInvites
      .map((inv) => inv.id)
      .filter(Boolean) as string[];
    if (inviteIds.length > 0) {
      markInvitesAsSeen(inviteIds);
    }
    close();
  }, [pendingInvites, close]);
  /* eslint-enable react-hooks/preserve-manual-memoization */

  // Auto-open modal once if there are unseen invites
  useEffect(() => {
    if (!isLoading && hasUnseenInvites && !hasAutoOpened.current && !opened) {
      hasAutoOpened.current = true;
      open();
    }
  }, [isLoading, hasUnseenInvites, opened, open]);

  const translateRole = useTranslateRole();

  const isWorkshopInvite = (invite: InviteCardView) =>
    !!invite.workshopId;

  const getInviteTitle = (invite: InviteCardView) => {
    if (isWorkshopInvite(invite)) {
      return t("notifications.workshopInvite");
    }
    return t("notifications.inviteToJoin");
  };

  const getInviteTarget = (invite: InviteCardView) => {
    if (isWorkshopInvite(invite)) {
      const wsName = invite.workshopName || invite.workshopId;
      const orgName = invite.institutionName;
      return orgName ? `${wsName} (${orgName})` : wsName;
    }
    return invite.institutionName || invite.institutionId || "-";
  };

  const handleAccept = (invite: InviteCardView) => {
    setAcceptingInvite(invite);
    acceptMutation.mutate(invite.id!);
  };

  const handleDecline = (invite: InviteCardView) => {
    setInviteToDecline(invite);
    openDeclineModal();
  };

  const confirmDecline = () => {
    if (inviteToDecline?.id) {
      declineMutation.mutate(inviteToDecline.id, {
        onSuccess: () => {
          closeDeclineModal();
          setInviteToDecline(null);
        },
      });
    }
  };

  // Admins have no invites; whoever may suspend organisations is told about
  // the organisations registered lately instead.
  if (standing.maySuspendInstitutions) {
    return <AdminRegistrationBell />;
  }
  if (isAdmin) {
    return null;
  }

  // The bell shows only while something waits; its dialogs stay until closed
  // (declining the last invite closes them itself).
  if (pendingCount === 0 && !opened && !declineModalOpened) {
    return null;
  }

  return (
    <>
      <Indicator
        inline
        label={pendingCount}
        size={18}
        offset={4}
        position="top-end"
        // white on the red bell; the theme's autoContrast writes the count dark
        color="white"
        disabled={pendingCount === 0}
      >
        <ActionIcon
          variant="filled"
          color="red"
          size="lg"
          radius="xl"
          onClick={open}
          aria-label={t("notifications.title")}
          className={pendingCount > 0 ? classes.ringing : undefined}
        >
          <IconBell size={22} />
        </ActionIcon>
      </Indicator>

      <Modal
        opened={opened}
        onClose={handleClose}
        title={t("notifications.title")}
        size="md"
      >
        {isLoading ? (
          <Center p="xl">
            <Loader size="sm" />
          </Center>
        ) : pendingInvites.length === 0 ? (
          <Text c="dimmed" ta="center" py="xl">
            {t("notifications.empty")}
          </Text>
        ) : (
          <Stack gap="md">
            {pendingInvites.map((invite: InviteCardView) => (
              <Card key={invite.id} withBorder padding="md">
                <Stack gap="sm">
                  <Group justify="space-between" align="flex-start">
                    <Stack gap={4}>
                      <Text fw={500}>{getInviteTitle(invite)}</Text>
                      <Text size="sm" fw={600}>
                        {getInviteTarget(invite)}
                      </Text>
                    </Stack>
                    {!isWorkshopInvite(invite) && (
                      <Badge variant="light">
                        {translateRole(invite.role)}
                      </Badge>
                    )}
                  </Group>
                  <Text size="sm" c="dimmed">
                    {isWorkshopInvite(invite)
                      ? t("notifications.workshopInviteDescription")
                      : t("notifications.inviteDescription", {
                          role: translateRole(invite.role),
                        })}
                  </Text>
                  <Group justify="flex-end" gap="xs">
                    <Button
                      variant="subtle"
                      color="gray"
                      size="xs"
                      onClick={() => handleDecline(invite)}
                    >
                      {t("notifications.decline")}
                    </Button>
                    <Button
                      size="xs"
                      onClick={() => handleAccept(invite)}
                      loading={acceptMutation.isPending}
                    >
                      {t("notifications.accept")}
                    </Button>
                  </Group>
                </Stack>
              </Card>
            ))}
          </Stack>
        )}
      </Modal>

      {/* Decline Confirmation Modal */}
      <Modal
        opened={declineModalOpened}
        onClose={() => {
          closeDeclineModal();
          setInviteToDecline(null);
        }}
        title={t("notifications.declineTitle")}
        size="sm"
      >
        <Stack gap="md">
          <Text>{t("notifications.declineConfirm")}</Text>
          <Group justify="flex-end" gap="xs">
            <Button
              variant="subtle"
              onClick={() => {
                closeDeclineModal();
                setInviteToDecline(null);
              }}
            >
              {t("cancel")}
            </Button>
            <Button
              color="red"
              onClick={confirmDecline}
              loading={declineMutation.isPending}
            >
              {t("notifications.decline")}
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
