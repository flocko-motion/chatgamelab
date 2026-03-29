/**
 * Help Pages Link Configuration
 *
 * Centralized configuration for all help/documentation page URLs used
 * throughout the application. This makes it easy to update the base URL
 * or individual paths when the documentation site changes.
 *
 * Usage:
 *   import { HELP_LINKS, getHelpUrl } from '@/config/helpLinks';
 *   const url = getHelpUrl(HELP_LINKS.TERMS_OF_SERVICE);
 */

/**
 * Base URL for the help/documentation site.
 * Change this to point to a different docs deployment.
 */
const HELP_BASE_URL = "https://docs.chatgamelab.eu";

/**
 * Contact email addresses used throughout the application.
 */
export const CONTACT_EMAILS = {
  /** General support / contact email */
  SUPPORT: "support@chatgamelab.eu",
  /** Email for organization registration requests (educators / professionals) */
  ORGANIZATION_REQUEST: "organizations@chatgamelab.eu",
} as const;

/**
 * Help page paths. Add new entries as pages become available.
 * Use descriptive const names so it's clear what each link points to.
 */
export const HELP_LINKS = {
  /** Terms of service / usage agreement */
  TERMS_OF_SERVICE: "https://chatgamelab.eu/nutzungsbedingungen/",
} as const;

/** Type for any help link path */
export type HelpLinkPath = (typeof HELP_LINKS)[keyof typeof HELP_LINKS];

/**
 * Build a full help page URL from a path constant.
 * Absolute URLs are returned as-is; relative paths are prefixed with HELP_BASE_URL.
 */
export function getHelpUrl(path: HelpLinkPath): string {
  if (path.startsWith("http://") || path.startsWith("https://")) {
    return path;
  }
  const base = HELP_BASE_URL.replace(/\/$/, "");
  const cleanPath = path.startsWith("/") ? path : `/${path}`;
  return `${base}${cleanPath}`;
}

/**
 * Get the base URL for the help site (useful for linking to the docs root).
 */
export function getHelpBaseUrl(): string {
  return HELP_BASE_URL;
}
