/**
 * Help Pages Link Configuration
 *
 * Centralized configuration for all help/documentation page URLs used
 * throughout the application. This makes it easy to update the base URL
 * or individual paths when the documentation site changes.
 *
 * Usage:
 *   import { HELP_LINKS, getHelpUrl } from '@/config/helpLinks';
 *   const url = getHelpUrl(HELP_LINKS.GETTING_STARTED);
 *   // => "https://docs.chatgamelab.eu/getting-started"
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
 * All help page paths, grouped by area.
 * Use descriptive const names so it's clear what each link points to.
 */
export const HELP_LINKS = {
  // ── Legal ──────────────────────────────────────────────────────────
  /** Privacy policy page */
  PRIVACY_POLICY: "/legal/privacy-policy",
  /** Terms of service / usage agreement (external, not on docs site) */
  TERMS_OF_SERVICE: "https://chatgamelab.eu/nutzungsbedingungen/",
  /** Imprint / legal notice */
  IMPRINT: "/legal/imprint",

  // ── Getting Started ────────────────────────────────────────────────
  /** Overview / onboarding guide */
  GETTING_STARTED: "/getting-started",
  /** How to create your first game */
  CREATE_FIRST_GAME: "/getting-started/create-first-game",
  /** Understanding the dashboard */
  DASHBOARD_OVERVIEW: "/getting-started/dashboard",

  // ── Game Creation ──────────────────────────────────────────────────
  /** How the game editor works */
  GAME_EDITOR: "/games/editor",
  /** Writing effective game prompts */
  GAME_PROMPTS: "/games/prompts",
  /** Game design / theme settings */
  GAME_DESIGN: "/games/design",
  /** Sharing games via links */
  GAME_SHARING: "/games/sharing",

  // ── Workshops ──────────────────────────────────────────────────────
  /** Workshop feature overview */
  WORKSHOPS_OVERVIEW: "/workshops/overview",
  /** Creating and managing workshops */
  WORKSHOP_MANAGEMENT: "/workshops/management",
  /** Workshop settings explained */
  WORKSHOP_SETTINGS: "/workshops/settings",
  /** How participants join workshops */
  WORKSHOP_JOINING: "/workshops/joining",

  // ── Organizations ──────────────────────────────────────────────────
  /** Organization management overview */
  ORGANIZATIONS_OVERVIEW: "/organizations/overview",
  /** Managing organization members */
  ORGANIZATION_MEMBERS: "/organizations/members",
  /** API key management */
  API_KEYS: "/organizations/api-keys",

  // ── Account ────────────────────────────────────────────────────────
  /** Profile settings help */
  PROFILE_SETTINGS: "/account/profile",
  /** Application settings help */
  APP_SETTINGS: "/account/settings",

  // ── AI & Models ────────────────────────────────────────────────────
  /** How AI models are used in the platform */
  AI_OVERVIEW: "/ai/overview",
  /** Configuring AI quality tiers */
  AI_QUALITY_TIERS: "/ai/quality-tiers",
} as const;

/** Type for any help link path */
export type HelpLinkPath = (typeof HELP_LINKS)[keyof typeof HELP_LINKS];

/**
 * Build a full help page URL from a path constant.
 *
 * @param path - One of the HELP_LINKS constants (e.g. HELP_LINKS.GETTING_STARTED)
 * @returns Full URL (e.g. "https://docs.chatgamelab.eu/getting-started")
 *
 * @example
 *   <a href={getHelpUrl(HELP_LINKS.PRIVACY_POLICY)}>Privacy Policy</a>
 */
export function getHelpUrl(path: HelpLinkPath): string {
  // If path is already an absolute URL, return it directly
  if (path.startsWith("http://") || path.startsWith("https://")) {
    return path;
  }
  // Strip trailing slash from base and leading slash from path to avoid doubles
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
