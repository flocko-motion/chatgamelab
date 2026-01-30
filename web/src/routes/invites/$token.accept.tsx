import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { Container, Stack, Card, Center, Loader, Alert, Title, Text, Group } from '@mantine/core';
import { IconAlertCircle, IconCheck, IconArrowRight } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useState, useEffect } from 'react';
import { ActionButton } from '@/common/components/buttons/ActionButton';
import { TextButton } from '@/common/components/buttons/TextButton';
import { config } from '@/config/env';
import { useAuth } from '@/providers/AuthProvider';
import { ROUTES } from '@/common/routes/routes';

export const Route = createFileRoute('/invites/$token/accept')({
  component: AcceptInvitePage,
});

interface InviteDetails {
  id: string;
  workshopId?: string;
  workshopName?: string;
  institutionName?: string;
  role?: string;
  status?: string;
}

function AcceptInvitePage() {
  const { t } = useTranslation('common');
  const { token } = Route.useParams();
  const navigate = useNavigate();
  const { isParticipant, backendUser, isLoading: authLoading } = useAuth();

  const [state, setState] = useState<'loading' | 'ready' | 'accepting' | 'success' | 'error' | 'switch-workshop'>('loading');
  const [invite, setInvite] = useState<InviteDetails | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isAccepting, setIsAccepting] = useState(false);

  // Get current workshop info if user is a participant
  const currentWorkshopId = backendUser?.role?.workshop?.id;
  const currentWorkshopName = backendUser?.role?.workshop?.name;

  // Fetch invite details on mount
  useEffect(() => {
    // Wait for auth to finish loading
    if (authLoading) return;

    async function fetchInvite() {
      try {
        const response = await fetch(`${config.API_BASE_URL}/invites/${token}`, {
          credentials: 'include',
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          if (response.status === 404) {
            setError(t('invites.errors.notFound'));
          } else if (errorData.code === 'invite_expired') {
            setError(t('invites.errors.expired'));
          } else {
            setError(t('invites.errors.loadFailed'));
          }
          setState('error');
          return;
        }

        const data = await response.json();
        setInvite(data);

        // Check if user is already in this workshop
        if (isParticipant && currentWorkshopId && data.workshopId === currentWorkshopId) {
          // Already in this workshop - redirect directly
          navigate({ to: ROUTES.MY_WORKSHOP as '/' });
          return;
        }

        // Check if user is in a different workshop
        if (isParticipant && currentWorkshopId && data.workshopId !== currentWorkshopId) {
          // Different workshop - show switch confirmation
          setState('switch-workshop');
          return;
        }

        setState('ready');
      } catch {
        setError(t('invites.errors.loadFailed'));
        setState('error');
      }
    }

    fetchInvite();
  }, [token, t, authLoading, isParticipant, currentWorkshopId, navigate]);

  const handleAccept = async () => {
    setIsAccepting(true);
    setState('accepting');

    try {
      const response = await fetch(`${config.API_BASE_URL}/invites/${token}/accept`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({}),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        setError(errorData.message || t('invites.errors.acceptFailed'));
        setState('error');
        return;
      }

      await response.json();

      // If we got an auth token, the cookie should already be set by the server
      setState('success');

      // Short delay to show success message, then redirect to my-workshop
      setTimeout(() => {
        // Force page reload to refresh auth state
        window.location.href = ROUTES.MY_WORKSHOP;
      }, 1500);
    } catch {
      setError(t('invites.errors.acceptFailed'));
      setState('error');
    }
  };

  if (state === 'loading') {
    return (
      <Container size="sm" py="xl">
        <Center py="xl">
          <Loader size="lg" />
        </Center>
      </Container>
    );
  }

  if (state === 'error') {
    return (
      <Container size="sm" py="xl">
        <Card shadow="sm" padding="xl" radius="md" withBorder>
          <Stack align="center" gap="md">
            <Alert
              icon={<IconAlertCircle size={16} />}
              title={t('error')}
              color="red"
              w="100%"
            >
              {error}
            </Alert>
            <ActionButton onClick={() => navigate({ to: '/' })}>
              {t('invites.goHome')}
            </ActionButton>
          </Stack>
        </Card>
      </Container>
    );
  }

  if (state === 'success') {
    return (
      <Container size="sm" py="xl">
        <Card shadow="sm" padding="xl" radius="md" withBorder>
          <Stack align="center" gap="md">
            <IconCheck size={48} color="var(--mantine-color-green-6)" />
            <Title order={2} ta="center">{t('invites.success.title')}</Title>
            <Text ta="center" c="dimmed">
              {t('invites.success.message')}
            </Text>
            <Loader size="sm" />
          </Stack>
        </Card>
      </Container>
    );
  }

  // Show switch workshop confirmation
  if (state === 'switch-workshop') {
    return (
      <Container size="sm" py="xl">
        <Card shadow="sm" padding="xl" radius="md" withBorder>
          <Stack gap="lg">
            <Title order={2} ta="center">
              {t('invites.switchWorkshop.title')}
            </Title>

            <Alert
              icon={<IconAlertCircle size={16} />}
              color="yellow"
            >
              {t('invites.switchWorkshop.warning', {
                currentWorkshop: currentWorkshopName,
                newWorkshop: invite?.workshopName,
              })}
            </Alert>

            <Text ta="center" c="dimmed">
              {t('invites.switchWorkshop.message')}
            </Text>

            <Group justify="center" gap="md">
              <TextButton onClick={() => navigate({ to: ROUTES.MY_WORKSHOP as '/' })}>
                {t('invites.switchWorkshop.stayButton')}
              </TextButton>
              <ActionButton
                onClick={handleAccept}
                loading={isAccepting}
                rightSection={<IconArrowRight size={16} />}
              >
                {t('invites.switchWorkshop.switchButton')}
              </ActionButton>
            </Group>
          </Stack>
        </Card>
      </Container>
    );
  }

  return (
    <Container size="sm" py="xl">
      <Card shadow="sm" padding="xl" radius="md" withBorder>
        <Stack gap="lg">
          <Title order={2} ta="center">
            {invite?.workshopName 
              ? t('invites.accept.titleWithWorkshop', { workshop: invite.workshopName })
              : t('invites.accept.title')
            }
          </Title>

          <Text ta="center" c="dimmed">
            {invite?.institutionName
              ? t('invites.accept.inviteMessage', { organization: invite.institutionName })
              : t('invites.accept.inviteMessageSimple')
            }
          </Text>

          <ActionButton
            onClick={handleAccept}
            loading={state === 'accepting'}
            fullWidth
          >
            {t('invites.accept.joinButton')}
          </ActionButton>
        </Stack>
      </Card>
    </Container>
  );
}
