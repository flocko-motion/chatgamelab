/**
 * External Links Configuration
 * 
 * Centralized configuration for external links used throughout the application.
 * This makes it easy to update URLs and descriptions in one place.
 */

export interface ExternalLink {
  id: string;
  title: string;
  description: string;
  href: string;
}

export const EXTERNAL_LINKS: Record<string, ExternalLink> = {
  CHATGAMELAB: {
    id: 'chatgamelab',
    title: 'ChatGameLab Website',
    description: 'Learn more about ChatGameLab and its educational mission',
    href: 'https://chatgamelab.eu',
  },
  CONTACT: {
    id: 'contact',
    title: 'Contact',
    description: 'Get in touch with the ChatGameLab team',
    href: 'https://chatgamelab.eu/?page_id=3',
  },
  JFF: {
    id: 'jff',
    title: 'JFF Institute',
    description: 'Discover the JFF - Institute for Educational Research and Innovation',
    href: 'https://jff.de',
  },
} as const;

export type ExternalLinkId = keyof typeof EXTERNAL_LINKS;
