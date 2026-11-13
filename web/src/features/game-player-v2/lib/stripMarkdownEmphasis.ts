/**
 * Removes paired markdown emphasis markers (**bold**, *italic*) on one line but keeps the
 * text between them. The narration prompt forbids markdown, but some models (mostly Mistral)
 * still use it; the player shows plain text through character-based text effects.
 * Lone stars or spaced ones ("5 * 3") stay. Mirrors stripMarkdownEmphasis in
 * server/game/narration_clean.go, which cleans the stored text; this one also covers the
 * live stream and messages saved before the server clean-up existed.
 */
const BOLD = /\*\*([^\s*](?:[^*\n]*[^\s*])?)\*\*/g;
const ITALIC = /\*([^\s*](?:[^*\n]*[^\s*])?)\*/g;

export function stripMarkdownEmphasis(text: string): string {
  return text.replace(BOLD, "$1").replace(ITALIC, "$1");
}
