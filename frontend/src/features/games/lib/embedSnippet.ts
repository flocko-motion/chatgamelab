/** Where the styling contract for embedded games is documented (#340's docs/embed-styling.md). */
export const EMBED_STYLING_DOCS_URL =
  "https://github.com/flocko-motion/chatgamelab/blob/v2/development/docs/embed-styling.md";

import type { Mode } from "@/api/types";

/** The frame's height the snippet starts with; the host changes it. */
const SNIPPET_HEIGHT = "32rem";

export interface SnippetParts {
  /** The playback-engine's loader script. */
  loaderUrl: string;
  token: string;
  /** For a verified page: the host fills in its visitor's and page view's IDs. */
  verified: boolean;
  /** The words standing in for what the host fills in, in the designer's language. */
  placeholders: { css: string; player: string; request: string };
  /** Live or rounds, for a game that plays both; none lets the player choose. */
  mode?: Mode;
}

/**
 * The snippet a host page holds: a style block for its CSS, the element the
 * game fills and the loader script. A verified page also passes its
 * visitor's and page view's IDs, which its callback gets.
 */
export function embedSnippet({ loaderUrl, token, verified, placeholders, mode }: SnippetParts): string {
  const ids = verified ? ` data-player="${placeholders.player}" data-request="${placeholders.request}"` : "";
  const half = mode ? ` data-mode="${mode}"` : "";
  return [
    "<style data-cgl>",
    `  /* ${placeholders.css} */`,
    "</style>",
    `<div data-cgl-embed data-token="${token}"${ids}${half} style="height: ${SNIPPET_HEIGHT}"></div>`,
    `<script src="${loaderUrl}" async></script>`,
  ].join("\n");
}
