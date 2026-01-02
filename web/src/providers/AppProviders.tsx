import { MantineProvider } from '@mantine/core';
import { ModalsProvider } from '@mantine/modals';
import { Notifications } from '@mantine/notifications';
import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { RouterProvider } from '@tanstack/react-router';

import { mantineTheme } from '../config/mantineTheme';
import { queryClient } from '../config/queryClient';
import { router } from '../config/router';

export function AppProviders() {
  return (
    <QueryClientProvider client={queryClient}>
      <MantineProvider theme={mantineTheme}>
        <ModalsProvider>
          <Notifications />
          <RouterProvider router={router} />
          <ReactQueryDevtools initialIsOpen={false} />
        </ModalsProvider>
      </MantineProvider>
    </QueryClientProvider>
  );
}
