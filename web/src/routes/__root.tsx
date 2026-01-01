import { createRootRoute, Outlet } from '@tanstack/react-router';
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools';
import { AppShell, Container } from '@mantine/core';

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return (
    <AppShell header={{ height: 60 }} padding="md">
      <AppShell.Header>
        <Container size="xl" h="100%" style={{ display: 'flex', alignItems: 'center' }}>
          <h1 style={{ margin: 0, fontSize: '1.5rem' }}>🎮 ChatGameLab</h1>
        </Container>
      </AppShell.Header>

      <AppShell.Main>
        <Container size="xl">
          <Outlet />
        </Container>
      </AppShell.Main>

      <TanStackRouterDevtools position="bottom-right" />
    </AppShell>
  );
}
