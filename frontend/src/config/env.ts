/**
 * The instance's public configuration, from the server's PUBLIC_* environment
 * variables (schemaf's runtime config, loaded before the app renders).
 *
 * Every value here is readable by the browser. Never put a secret here.
 */
import { config as runtimeConfig } from '@/api/config';

type AppConfig = {
  /** Auth0 tenant domain; with AUTH0_CLIENT_ID, Auth0 sign-in is offered. */
  AUTH0_DOMAIN?: string;
  AUTH0_CLIENT_ID?: string;
  /** The app's public URL, the base of share links. Defaults to the page's origin. */
  PUBLIC_URL: string;
  SENTRY_DSN?: string;
  /**
   * External homepage URL for deployments with a separate landing page.
   * When set, the root route (/) redirects here, and logout flows use this URL.
   * When not set, the built-in landing page is shown at /.
   */
  HOMEPAGE_URL?: string;
};

const value = (key: string) => runtimeConfig(key) || undefined;

export const config: AppConfig = {
  AUTH0_DOMAIN: value('PUBLIC_AUTH0_DOMAIN'),
  AUTH0_CLIENT_ID: value('PUBLIC_AUTH0_CLIENT_ID'),
  PUBLIC_URL: value('PUBLIC_URL') ?? '/',
  SENTRY_DSN: value('PUBLIC_SENTRY_DSN'),
  HOMEPAGE_URL: value('PUBLIC_HOMEPAGE_URL'),
};

const env = {
  MODE: import.meta.env.MODE,
  DEV: import.meta.env.DEV,
  PROD: import.meta.env.PROD,
  BASE_URL: import.meta.env.BASE_URL,
} as const;

export default env;
