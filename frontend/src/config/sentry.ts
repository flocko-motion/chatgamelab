import * as Sentry from '@sentry/react';
import { config } from './env';

export function initSentry() {
  const dsn = config.SENTRY_DSN_FRONTEND;
  if (!dsn) {
    return;
  }

  Sentry.init({
    dsn,
    environment: import.meta.env.MODE,
    enabled: !!dsn,
    // Send 100% of errors, sample 10% of performance traces
    sampleRate: 1.0,
    tracesSampleRate: 0.1,
  });
}
