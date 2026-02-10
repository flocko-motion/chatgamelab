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
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useRequiredAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { useAuth } from "@/providers/AuthProvider";
import { queryKeys } from "@/api/hooks";
import type { RoutesInviteResponse } from "@/api/generated";

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
  const { t: tAuth } = useTranslation("auth");
  const api = useRequiredAuthenticatedApi();
  const { backendUser, retryBackendFetch } = useAuth();
  const queryClient = useQueryClient();
  const [opened, { open, close }] = useDisclosure(false);
  const [
    declineModalOpened,
    { open: openDeclineModal, close: closeDeclineModal },
  ] = useDisclosure(false);
  const [inviteToDecline, setInviteToDecline] =
    useState<RoutesInviteResponse | null>(null);
  const hasAutoOpened = useRef(false);

  // Admins don't have invites - disable query and hide the bell
  const isAdmin = backendUser?.role?.role === "admin";

  const { data: invites, isLoading } = useQuery({
    queryKey: queryKeys.invites,
    queryFn: async () => {
      const response = await api.invites.invitesList();
      return response.data;
    },
    enabled: !isAdmin, // Don't fetch invites for admins
  });

  const navigate = useNavigate();

  const acceptMutation = useMutation({
    mutationFn: async (inviteId: string) => {
      const response = await api.invites.acceptCreate(inviteId);
      return response.data;
    },
    onSuccess: () => {
      // Mark all invites as seen to prevent auto-reopen
      const inviteIds = pendingInvites
        .map((inv) => inv.id)
        .filter(Boolean) as string[];
      if (inviteIds.length > 0) {
        markInvitesAsSeen(inviteIds);
      }

      queryClient.refetchQueries({ queryKey: queryKeys.invites });
      queryClient.refetchQueries({ queryKey: queryKeys.currentUser });
      retryBackendFetch(); // Refresh user's organization data
      close(); // Close the notifications modal
      // Delay navigation slightly to ensure modal closes
      setTimeout(() => {
        navigate({ to: "/my-organization" });
      }, 100);
    },
  });

  const declineMutation = useMutation({
    mutationFn: async (inviteId: string) => {
      const response = await api.invites.declineCreate(inviteId);
      return response.data;
    },
    onSuccess: () => {
      queryClient.refetchQueries({ queryKey: queryKeys.invites });
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

  // Fetch organization and workshop names for invites
  const [orgNames, setOrgNames] = useState<Record<string, string>>({});
  const [workshopNames, setWorkshopNames] = useState<Record<string, string>>(
    {},
  );
  // Track IDs we've already fetched to avoid re-fetching (and infinite loops)
  const fetchedOrgIdsRef = useRef<Set<string>>(new Set());
  const fetchedWorkshopIdsRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    const fetchNames = async () => {
      const newOrgNames: Record<string, string> = {};
      const newWorkshopNames: Record<string, string> = {};

      for (const invite of pendingInvites) {
        if (
          invite.institutionId &&
          !fetchedOrgIdsRef.current.has(invite.institutionId)
        ) {
          fetchedOrgIdsRef.current.add(invite.institutionId);
          try {
            const response = await api.institutions.institutionsDetail(
              invite.institutionId,
            );
            newOrgNames[invite.institutionId] =
              response.data.name || invite.institutionId;
          } catch {
            newOrgNames[invite.institutionId] = invite.institutionId;
          }
        }
        if (
          invite.workshopId &&
          !fetchedWorkshopIdsRef.current.has(invite.workshopId)
        ) {
          fetchedWorkshopIdsRef.current.add(invite.workshopId);
          try {
            const response = await api.workshops.workshopsDetail(
              invite.workshopId,
            );
            newWorkshopNames[invite.workshopId] =
              response.data.name || invite.workshopId;
          } catch {
            newWorkshopNames[invite.workshopId] = invite.workshopId;
          }
        }
      }

      if (Object.keys(newOrgNames).length > 0) {
        setOrgNames((prev) => ({ ...prev, ...newOrgNames }));
      }
      if (Object.keys(newWorkshopNames).length > 0) {
        setWorkshopNames((prev) => ({ ...prev, ...newWorkshopNames }));
      }
    };

    if (pendingInvites.length > 0) {
      fetchNames();
    }
  }, [pendingInvites, api]);

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

  const translateRole = (role?: string) => {
    if (!role) return "-";
    const roleKey = role.toLowerCase();
    return tAuth(`profile.roles.${roleKey}`, role);
  };

  const getInviteTarget = (invite: RoutesInviteResponse) => {
    if (invite.workshopId) {
      return workshopNames[invite.workshopId] || invite.workshopId;
    }
    if (invite.institutionId) {
      return orgNames[invite.institutionId] || invite.institutionId;
    }
    return "-";
  };

  const handleAccept = (inviteId: string) => {
    acceptMutation.mutate(inviteId);
  };

  const handleDecline = (invite: RoutesInviteResponse) => {
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

  // Hide bell for admin users
  if (isAdmin) {
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
        color="red"
        disabled={pendingCount === 0}
        processing={pendingCount > 0}
      >
        <ActionIcon
          variant="subtle"
          color="white"
          size="lg"
          onClick={open}
          aria-label={t("notifications.title")}
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
            {pendingInvites.map((invite: RoutesInviteResponse) => (
              <Card key={invite.id} withBorder padding="md">
                <Stack gap="sm">
                  <Group justify="space-between" align="flex-start">
                    <Stack gap={4}>
                      <Text fw={500}>{t("notifications.inviteToJoin")}</Text>
                      <Text size="sm" fw={600}>
                        {getInviteTarget(invite)}
                      </Text>
                    </Stack>
                    <Badge variant="light">{translateRole(invite.role)}</Badge>
                  </Group>
                  <Text size="sm" c="dimmed">
                    {t("notifications.inviteDescription", {
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
                      onClick={() => handleAccept(invite.id!)}
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
