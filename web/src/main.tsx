import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import '@mantine/core/styles.css';
import '@mantine/dates/styles.css';
import '@mantine/notifications/styles.css';
import { AppProviders } from './providers/AppProviders';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppProviders />
  </StrictMode>,
);
