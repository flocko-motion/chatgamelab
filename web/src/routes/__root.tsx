import { createRootRoute, Outlet } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { AppShell, Container, Text, Group, Anchor, Box } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import { LanguageSwitcher } from '@components/LanguageSwitcher';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  const { t } = useTranslation('common');

  return (
    <AppShell footer={{ height: 50 }} padding="md">
      <Box style={{ position: 'fixed', top: 16, right: 16, zIndex: 1000 }}>
        <LanguageSwitcher />
      </Box>

      <AppShell.Main>
        <Container size="xl">
          <Outlet />
        </Container>
      </AppShell.Main>

      <AppShell.Footer>
        <Container size="xl" h="100%">
          <Box 
            h="100%" 
            style={{ 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center' 
            }}
          >
            <Text size="sm" c="dimmed" component="div">
              <Group gap="sm">
                <Anchor href="https://auth0.com" target="_blank" size="sm">
                  {t('footer.loginByAuth0')}
                </Anchor>
                <Text size="sm" span>|</Text>
                <Anchor href="https://omnitopos.net" target="_blank" size="sm">
                  {t('footer.programmedByOmnitopos')}
                </Anchor>
                <Text size="sm" span>|</Text>
                <Anchor href="https://tausend-medien.de" target="_blank" size="sm">
                  {t('footer.producedByTausendMedien')}
                </Anchor>
                <Text size="sm" span>|</Text>
                <Text size="sm" span>{t('footer.version')}</Text>
              </Group>
            </Text>
          </Box>
        </Container>
      </AppShell.Footer>

      <TanStackRouterDevtools position="bottom-right" />
    </AppShell>
  );
}
