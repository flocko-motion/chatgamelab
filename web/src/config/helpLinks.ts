/**
 * Help Pages Link Configuration
 *
 * Centralized configuration for all help/documentation page URLs used
 * throughout the application.
 *
 * Usage:
 *   import { HELP_LINKS } from '@/config/helpLinks';
 *   <a href={HELP_LINKS.TERMS_OF_SERVICE}>Terms</a>
 */

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
 * Help page URLs. Add new entries as pages become available.
 * Use descriptive const names so it's clear what each link points to.
 */
export const HELP_LINKS = {
  /** Terms of service / usage agreement */
  TERMS_OF_SERVICE: "https://chatgamelab.eu/nutzungsbedingungen/",
  /** Information about using ChatGameLab for educational purposes */
  EDUCATOR_INFO: "https://chatgamelab.eu/edu/",
} as const;
