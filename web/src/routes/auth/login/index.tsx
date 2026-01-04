import { createFileRoute, useRouter } from '@tanstack/react-router';
import { Button, Container, Paper, Stack, Title, Text, Divider } from '@mantine/core';
import { useAuth } from '@/providers/AuthProvider';
import { useTranslation } from 'react-i18next';
import { useEffect } from 'react';

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
      router.navigate({ to: '/dashboard' });
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
          <Title order={2}>{t('login.redirecting.title')}</Title>
          <Text c="dimmed">{t('login.redirecting.message')}</Text>
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
            <Title order={2} c="yellow">
              {t('login.devModeAlert.title')}
            </Title>
            <Text c="dimmed" ta="center" size="sm">
              {t('login.devModeAlert.message')}
            </Text>
          </Stack>

          <Button
            size="lg"
            onClick={loginWithAuth0}
            variant="filled"
            fullWidth
          >
            {t('login.auth0Button')}
          </Button>

          <Divider label={t('login.devMode')} labelPosition="center" />
          
          <Stack gap="sm">
            <Text size="sm" c="dimmed" ta="center">
              {t('login.devModeDescription')}
            </Text>
            
            {devRoles.map((role) => (
              <Button
                key={role.key}
                variant="outline"
                onClick={() => {
                  loginWithRole(role.key);
                  // Redirect to dashboard after login
                  setTimeout(() => {
                    router.navigate({ to: '/dashboard' });
                  }, 100);
                }}
                fullWidth
              >
                {t(`login.role.${role.key}`)} ({role.label})
              </Button>
            ))}
          </Stack>
        </Stack>
      </Paper>
    </Container>
  );
}
