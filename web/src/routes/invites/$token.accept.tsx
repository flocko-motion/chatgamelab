import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { Container, Stack, Card, Center, Loader, Alert, Title, Text, Group, Image } from '@mantine/core';
import { IconAlertCircle, IconCheck, IconArrowRight, IconLogout } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import { useState, useEffect, type ReactNode } from 'react';
import { ActionButton } from '@/common/components/buttons/ActionButton';
import { TextButton } from '@/common/components/buttons/TextButton';
import { config } from '@/config/env';
import { useAuth } from '@/providers/AuthProvider';
import { useWorkshopMode } from '@/providers/WorkshopModeProvider';
import { ROUTES } from '@/common/routes/routes';
import { buildShareUrl, getCookiePath } from '@/common/lib/url';
import logo from '@/assets/logos/colorful/ChatGameLab-Logo-2025-Square-Colorful2-Black-Text.png-Black-Text-Transparent.png';

export const Route = createFileRoute('/invites/$token/accept')({
  component: AcceptInvitePage,
});

function InvitePageLayout({ children, showBranding = true }: { children: ReactNode; showBranding?: boolean }) {
  const { t } = useTranslation('common');
  return (
    <Container size="sm" py="xl">
      <Stack gap="xl" align="center">
        {children}
        {showBranding && (
          <Stack gap="sm" align="center" ta="center" mt="md">
            <Image
              src={logo}
              alt="ChatGameLab Logo"
              w={{ base: 200, sm: 280, lg: 350 }}
              h={{ base: 200, sm: 280, lg: 350 }}
              fit="contain"
            />
            <Text size="sm" c="dimmed" maw={400}>
              {t('home.splashDescription')}
            </Text>
          </Stack>
        )}
      </Stack>
    </Container>
  );
}

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
  const { isParticipant, isAuthenticated, backendUser, isLoading: authLoading, logout } = useAuth();
  const { enterWorkshopMode, isLoading: isEnteringWorkshop } = useWorkshopMode();

  // Check if user is a non-participant authenticated user (admin, head, staff, individual)
  const isLoggedInNonParticipant = isAuthenticated && !isParticipant;

  const [state, setState] = useState<'loading' | 'ready' | 'accepting' | 'success' | 'error' | 'switch-workshop' | 'already-logged-in' | 'already-member'>('loading');
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
        const response = await fetch(`${config.API_BASE_URL}/invites/${token}`);

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

        // Check if user is logged in with a non-participant account
        if (isLoggedInNonParticipant) {
          // Check if user is head/staff of the organization that owns this workshop
          const userInstitutionId = backendUser?.role?.institution?.id;
          const userRole = backendUser?.role?.role;
          if (userInstitutionId && data.institutionId === userInstitutionId && (userRole === 'head' || userRole === 'staff')) {
            setState('already-member');
            return;
          }
          // Other authenticated users: show "already logged in" message
          setState('already-logged-in');
          return;
        }

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
  }, [token, t, authLoading, isParticipant, isLoggedInNonParticipant, backendUser, currentWorkshopId, navigate]);

  const handleAccept = async () => {
    setIsAccepting(true);
    setState('accepting');

    try {
      // Clear old session cookie first to prevent 401 from invalid token
      document.cookie = `cgl_session=; path=${getCookiePath()}; expires=Thu, 01 Jan 1970 00:00:00 GMT`;
      
      // Small delay to ensure cookie deletion is processed
      await new Promise(resolve => setTimeout(resolve, 50));
      
      // Now accept with credentials so browser saves the new cookie
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
        window.location.href = buildShareUrl(ROUTES.MY_WORKSHOP);
      }, 1500);
    } catch {
      setError(t('invites.errors.acceptFailed'));
      setState('error');
    }
  };

  if (state === 'loading') {
    return (
      <InvitePageLayout>
        <Center py="xl">
          <Loader size="lg" />
        </Center>
      </InvitePageLayout>
    );
  }

  if (state === 'error') {
    return (
      <InvitePageLayout>
        <Card shadow="sm" padding="xl" radius="md" withBorder w="100%">
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
      </InvitePageLayout>
    );
  }

  if (state === 'success') {
    return (
      <InvitePageLayout>
        <Card shadow="sm" padding="xl" radius="md" withBorder w="100%">
          <Stack align="center" gap="md">
            <IconCheck size={48} color="var(--mantine-color-green-6)" />
            <Title order={2} ta="center">{t('invites.success.title')}</Title>
            <Text ta="center" c="dimmed">
              {t('invites.success.message')}
            </Text>
            <Loader size="sm" />
          </Stack>
        </Card>
      </InvitePageLayout>
    );
  }

  // Show "already logged in" message for non-participant authenticated users
  if (state === 'already-logged-in') {
    return (
      <InvitePageLayout showBranding={false}>
        <Card shadow="sm" padding="xl" radius="md" withBorder w="100%">
          <Stack gap="lg">
            <Title order={2} ta="center">
              {t('invites.alreadyLoggedIn.title')}
            </Title>

            <Alert
              icon={<IconAlertCircle size={16} />}
              color="yellow"
              w="100%"
            >
              {t('invites.alreadyLoggedIn.message')}
            </Alert>

            <Group justify="center" gap="md">
              <TextButton onClick={() => navigate({ to: '/' })}>
                {t('invites.goHome')}
              </TextButton>
              <ActionButton
                onClick={() => logout()}
                leftSection={<IconLogout size={16} />}
              >
                {t('invites.alreadyLoggedIn.logoutButton')}
              </ActionButton>
            </Group>
          </Stack>
        </Card>
      </InvitePageLayout>
    );
  }

  // Show "already a member" message for head/staff of the owning org
  if (state === 'already-member') {
    const handleEnterWorkshop = async () => {
      if (invite?.workshopId && invite?.workshopName) {
        await enterWorkshopMode(invite.workshopId, invite.workshopName);
        navigate({ to: ROUTES.MY_WORKSHOP as '/' });
      }
    };

    return (
      <InvitePageLayout showBranding={false}>
        <Card shadow="sm" padding="xl" radius="md" withBorder w="100%">
          <Stack gap="lg">
            <Title order={2} ta="center">
              {t('invites.alreadyMember.title')}
            </Title>

            <Text ta="center" c="dimmed">
              {t('invites.alreadyMember.message')}
            </Text>

            <Group justify="center" gap="md">
              <TextButton onClick={() => navigate({ to: '/' })}>
                {t('invites.goHome')}
              </TextButton>
              <ActionButton
                onClick={handleEnterWorkshop}
                loading={isEnteringWorkshop}
                rightSection={<IconArrowRight size={16} />}
              >
                {t('invites.alreadyMember.enterButton')}
              </ActionButton>
            </Group>
          </Stack>
        </Card>
      </InvitePageLayout>
    );
  }

  // Show switch workshop confirmation
  if (state === 'switch-workshop') {
    return (
      <InvitePageLayout showBranding={false}>
        <Card shadow="sm" padding="xl" radius="md" withBorder w="100%">
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
      </InvitePageLayout>
    );
  }

  return (
    <InvitePageLayout>
      <Card shadow="sm" padding="xl" radius="md" withBorder w="100%">
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
    </InvitePageLayout>
  );
}
