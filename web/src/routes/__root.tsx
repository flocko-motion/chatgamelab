import { createRootRoute, Outlet } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { AppShell, Container, Text, Anchor, Box } from '@mantine/core';
import { useTranslation } from 'react-i18next';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  const { t } = useTranslation('common');

  return (
    <AppShell footer={{ height: { base: 80, sm: 60 } }} padding={{ base: 'sm', sm: 'md' }}>
      <AppShell.Main pt={{ base: 60, sm: 80 }}>
        <Container size="xl" px={{ base: 'sm', sm: 'md', lg: 'xl' }}>
          <Outlet />
        </Container>
      </AppShell.Main>

      <AppShell.Footer>
        <Container size="xl" h="100%" px={{ base: 'sm', sm: 'md', lg: 'xl' }}>
          <Box 
            h="100%" 
            style={{ 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center'
            }}
          >
            <Text size="sm" c="dimmed" component="div" ta="center">
              <Box style={{ display: 'flex', flexWrap: 'wrap', justifyContent: 'center', alignItems: 'center', gap: '8px' }}>
                <Text size="sm" c="dimmed" span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                  Login via <Anchor href="https://auth0.com" target="_blank" size="sm" c="violet">Auth0</Anchor>
                </Text>
                <Text size="sm" c="dimmed" span>|</Text>
                <Text size="sm" c="dimmed" span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                  Programmed by <Anchor href="https://omnitopos.net" target="_blank" size="sm" c="violet">omnitopos.net</Anchor>
                </Text>
                <Text size="sm" c="dimmed" span>|</Text>
                <Text size="sm" c="dimmed" span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                  Produced by <Anchor href="https://tausend-medien.de" target="_blank" size="sm" c="violet">tausend-medien.de</Anchor>
                </Text>
                <Text size="sm" c="dimmed" span>|</Text>
                <Text size="sm" c="dimmed" span>{t('footer.version')}</Text>
              </Box>
            </Text>
          </Box>
        </Container>
      </AppShell.Footer>

      <TanStackRouterDevtools position="bottom-right" />
    </AppShell>
  );
}
