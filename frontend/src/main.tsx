// Loads runtime config (/api/config) before the app renders. Keep it first.
import './api/generated/config.gen';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import '@mantine/core/styles.css';
import '@mantine/dates/styles.css';
import '@mantine/notifications/styles.css';
import './mobile-inputs.css';
import { initSentry } from './config/sentry';
import { AppProviders } from './providers/AppProviders';

initSentry();

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppProviders />
  </StrictMode>,
);
