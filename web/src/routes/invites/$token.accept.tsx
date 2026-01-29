import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { Container, Stack, Card, Center, Loader, Alert, Title, Text } from '@mantine/core';
import { IconAlertCircle, IconCheck } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useState, useEffect } from 'react';
import { ActionButton } from '@/common/components/buttons/ActionButton';
import { config } from '@/config/env';

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

  const [state, setState] = useState<'loading' | 'ready' | 'accepting' | 'success' | 'error'>('loading');
  const [invite, setInvite] = useState<InviteDetails | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Fetch invite details on mount
  useEffect(() => {
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
        setState('ready');
      } catch {
        setError(t('invites.errors.loadFailed'));
        setState('error');
      }
    }

    fetchInvite();
  }, [token, t]);

  const handleAccept = async () => {
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
      // Just navigate to dashboard
      setState('success');

      // Short delay to show success message, then redirect
      setTimeout(() => {
        navigate({ to: '/dashboard' });
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
