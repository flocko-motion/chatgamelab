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
  SUPPORT: "chatgamelab@jff.de",
  /** Email for organization registration requests (educators / professionals) */
  ORGANIZATION_REQUEST: "chatgamelab@jff.de",
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
  /** General help / documentation page */
  HELP_PAGE: "https://chatgamelab.eu/hilfe/",
  /** Tips for creating games */
  GAME_TIPS: "https://chatgamelab.eu/tipps/",
  /** API key FAQ */
  API_KEY_FAQ: "https://chatgamelab.eu/faq#api",
  /** AI insights explanation */
  AI_INSIGHTS: "https://chatgamelab.eu/ai-insights",
} as const;
