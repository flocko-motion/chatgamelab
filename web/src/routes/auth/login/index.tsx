import { createFileRoute, useRouter } from '@tanstack/react-router';
import { Container, Paper, Stack, Divider, Button as MantineButton } from '@mantine/core';
import { ActionButton } from '@components/buttons';
import { SectionTitle, HelperText } from '@components/typography';
import { useAuth } from '@/providers/AuthProvider';
import { useTranslation } from 'react-i18next';
import { useEffect } from 'react';
import { ROUTES } from '@/common/routes/routes';

export const Route = createFileRoute('/auth/login/')({
  component: LoginComponent,
});

function LoginComponent() {
  const { t } = useTranslation('auth');
  const { loginWithAuth0, loginWithRole, isDevMode, user } = useAuth();
  const router = useRouter();

  // Redirect to dashboard if already authenticated
  useEffect(() => {
    if (user) {
      router.navigate({ to: ROUTES.DASHBOARD });
    }
  }, [user, router]);

  // In production mode, redirect directly to Auth0
  useEffect(() => {
    if (!isDevMode) {
      loginWithAuth0();
    }
  }, [isDevMode, loginWithAuth0]);

  // Show loading while redirecting in production
  if (!isDevMode) {
    return (
      <Container size="sm" py="xl">
        <Stack gap="lg" align="center">
          <SectionTitle>{t('login.redirecting.title')}</SectionTitle>
          <HelperText>{t('login.redirecting.message')}</HelperText>
        </Stack>
      </Container>
    );
  }

  const devRoles = [
    { key: 'admin', label: 'Administrator' },
    { key: 'teacher', label: 'Teacher' },
    { key: 'student', label: 'Student' },
    { key: 'guest', label: 'Guest' },
  ];

  return (
    <Container size="sm" py="xl">
      <Paper shadow="md" p="xl" withBorder>
        <Stack gap="lg">
          <Stack gap="xs" align="center">
            <SectionTitle>
              {t('login.devModeAlert.title')}
            </SectionTitle>
            <HelperText>
              {t('login.devModeAlert.message')}
            </HelperText>
          </Stack>

          <ActionButton
            onClick={loginWithAuth0}
            fullWidth
          >
            {t('login.auth0Button')}
          </ActionButton>

          <Divider label={t('login.devMode')} labelPosition="center" />
          
          <Stack gap="sm">
            <HelperText>
              {t('login.devModeDescription')}
            </HelperText>
            
            {devRoles.map((role) => (
              <MantineButton
                key={role.key}
                variant="outline"
                color="gray"
                onClick={async () => {
                  await loginWithRole(role.key);
                  router.navigate({ to: ROUTES.DASHBOARD });
                }}
                fullWidth
              >
                {t(`login.role.${role.key}`)} ({role.label})
              </MantineButton>
            ))}
          </Stack>
        </Stack>
      </Paper>
    </Container>
  );
}
