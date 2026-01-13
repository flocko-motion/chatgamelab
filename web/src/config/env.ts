/**
 * Runtime configuration for the app.
 * 
 * In development: Uses VITE_* environment variables
 * In production: Uses window.__APP_CONFIG__ injected by container entrypoint
 * 
 * Note: All values here are PUBLIC (readable by browser). Never put secrets here.
 */

type AppConfig = {
  API_BASE_URL: string;
  AUTH0_DOMAIN: string;
  AUTH0_CLIENT_ID: string;
  AUTH0_AUDIENCE: string;
  AUTH0_REDIRECT_URI?: string;
  PUBLIC_URL?: string;
};

declare global {
  interface Window {
    __APP_CONFIG__?: Partial<AppConfig>;
  }
}

function must<T>(value: T | undefined, name: string): T {
  if (value === undefined || value === null || value === '') {
    throw new Error(`Missing required config: ${name}`);
  }
  return value;
}

const runtime = (typeof window !== 'undefined' && window.__APP_CONFIG__) || {};

export const config: AppConfig = {
  API_BASE_URL: must(runtime.API_BASE_URL ?? import.meta.env.VITE_API_BASE_URL, 'API_BASE_URL'),
  AUTH0_DOMAIN: must(runtime.AUTH0_DOMAIN ?? import.meta.env.VITE_AUTH0_DOMAIN, 'AUTH0_DOMAIN'),
  AUTH0_CLIENT_ID: must(runtime.AUTH0_CLIENT_ID ?? import.meta.env.VITE_AUTH0_CLIENT_ID, 'AUTH0_CLIENT_ID'),
  AUTH0_AUDIENCE: must(runtime.AUTH0_AUDIENCE ?? import.meta.env.VITE_AUTH0_AUDIENCE, 'AUTH0_AUDIENCE'),
  AUTH0_REDIRECT_URI: runtime.AUTH0_REDIRECT_URI ?? import.meta.env.VITE_AUTH0_REDIRECT_URI,
  // PUBLIC_URL is set at build time via Vite's base config, available at runtime via import.meta.env.BASE_URL
  PUBLIC_URL: runtime.PUBLIC_URL || import.meta.env.BASE_URL || '/',
};

const env = {
  MODE: import.meta.env.MODE,
  DEV: import.meta.env.DEV,
  PROD: import.meta.env.PROD,
  BASE_URL: import.meta.env.BASE_URL,
} as const;

export default env;
