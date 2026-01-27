import { useEffect, useRef, useCallback, useMemo } from 'react';
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
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconBell } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useRequiredAuthenticatedApi } from '@/api/useAuthenticatedApi';
import { useAuth } from '@/providers/AuthProvider';
import { queryKeys } from '@/api/hooks';
import type { RoutesInviteResponse } from '@/api/generated';

const SEEN_INVITES_KEY = 'cgl_seen_invites';

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
  const { t } = useTranslation('common');
  const { t: tAuth } = useTranslation('auth');
  const api = useRequiredAuthenticatedApi();
  const { backendUser } = useAuth();
  const queryClient = useQueryClient();
  const [opened, { open, close }] = useDisclosure(false);
  const hasAutoOpened = useRef(false);

  // Admins don't have invites - disable query and hide the bell
  const isAdmin = backendUser?.role?.role === 'admin';

  const { data: invites, isLoading } = useQuery({
    queryKey: queryKeys.invites,
    queryFn: async () => {
      const response = await api.invites.invitesList();
      return response.data;
    },
    enabled: !isAdmin, // Don't fetch invites for admins
  });

  const acceptMutation = useMutation({
    mutationFn: async (inviteId: string) => {
      const response = await api.invites.acceptCreate(inviteId);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.invites });
      queryClient.invalidateQueries({ queryKey: queryKeys.currentUser });
    },
  });

  const declineMutation = useMutation({
    mutationFn: async (inviteId: string) => {
      const response = await api.invites.declineCreate(inviteId);
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.invites });
    },
  });

  const pendingInvites = useMemo(
    () => invites?.filter((inv) => inv.status === 'pending') || [],
    [invites]
  );
  const pendingCount = pendingInvites.length;

  // Check for unseen invites
  const seenIds = getSeenInviteIds();
  const unseenInvites = pendingInvites.filter((inv) => inv.id && !seenIds.has(inv.id));
  const hasUnseenInvites = unseenInvites.length > 0;

  // Mark invites as seen when modal closes
  const handleClose = useCallback(() => {
    const inviteIds = pendingInvites.map((inv) => inv.id).filter(Boolean) as string[];
    if (inviteIds.length > 0) {
      markInvitesAsSeen(inviteIds);
    }
    close();
  }, [pendingInvites, close]);

  // Auto-open modal once if there are unseen invites
  useEffect(() => {
    if (!isLoading && hasUnseenInvites && !hasAutoOpened.current && !opened) {
      hasAutoOpened.current = true;
      open();
    }
  }, [isLoading, hasUnseenInvites, opened, open]);

  const translateRole = (role?: string) => {
    if (!role) return '-';
    const roleKey = role.toLowerCase();
    return tAuth(`profile.roles.${roleKey}`, role);
  };

  const handleAccept = (inviteId: string) => {
    acceptMutation.mutate(inviteId);
  };

  const handleDecline = (inviteId: string) => {
    if (confirm(t('notifications.declineConfirm'))) {
      declineMutation.mutate(inviteId);
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
          aria-label={t('notifications.title')}
        >
          <IconBell size={22} />
        </ActionIcon>
      </Indicator>

      <Modal
        opened={opened}
        onClose={handleClose}
        title={t('notifications.title')}
        size="md"
      >
        {isLoading ? (
          <Center p="xl">
            <Loader size="sm" />
          </Center>
        ) : pendingInvites.length === 0 ? (
          <Text c="dimmed" ta="center" py="xl">
            {t('notifications.empty')}
          </Text>
        ) : (
          <Stack gap="md">
            {pendingInvites.map((invite: RoutesInviteResponse) => (
              <Card key={invite.id} withBorder padding="md">
                <Stack gap="sm">
                  <Group justify="space-between">
                    <Text fw={500}>{t('notifications.inviteToJoin')}</Text>
                    <Badge variant="light">{translateRole(invite.role)}</Badge>
                  </Group>
                  <Text size="sm" c="dimmed">
                    {t('notifications.inviteDescription', {
                      organization: invite.institutionId,
                      role: translateRole(invite.role),
                    })}
                  </Text>
                  <Group justify="flex-end" gap="xs">
                    <Button
                      variant="subtle"
                      color="gray"
                      size="xs"
                      onClick={() => handleDecline(invite.id!)}
                      loading={declineMutation.isPending}
                    >
                      {t('notifications.decline')}
                    </Button>
                    <Button
                      size="xs"
                      onClick={() => handleAccept(invite.id!)}
                      loading={acceptMutation.isPending}
                    >
                      {t('notifications.accept')}
                    </Button>
                  </Group>
                </Stack>
              </Card>
            ))}
          </Stack>
        )}
      </Modal>
    </>
  );
}
