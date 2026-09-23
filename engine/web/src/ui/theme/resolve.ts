/**
 * Turns a theme into what the page applies: a complete theme merged over the
 * defaults, and the CSS custom properties the stylesheet reads.
 *
 * Ported from the platform's player (game-player-v2/theme) so a game looks the
 * same in both. Pure functions: nothing here touches the DOM.
 */
import {
  CARD_BG_COLORS,
  CARD_BORDER_THICKNESSES,
  DEFAULT_GAME_THEME,
  FONT_COLORS,
  THEME_COLORS,
  THEME_FONTS,
} from "./defaults.js";
import { PRESETS } from "./presets/index.js";
import type { BackgroundAnimation, BackgroundTint, GameTheme, PartialGameTheme } from "./types.js";

/**
 * What the platform's theme generator produces for a game: a preset picked by
 * a model, and a few overrides on top of it.
 */
export interface GeneratedTheme {
  readonly preset?: string;
  readonly animation?: string;
  readonly thinkingText?: string;
  readonly statusEmojis?: Record<string, string>;
}

/** A generated theme as a partial one. An unknown preset falls back to the default. */
export function fromGenerated(generated: GeneratedTheme): PartialGameTheme {
  const preset = PRESETS[generated.preset ?? ""] ?? PRESETS["default"];
  const result: PartialGameTheme = structuredClone(preset?.theme ?? {});
  if (generated.animation) {
    result.background = { ...result.background, animation: generated.animation as BackgroundAnimation };
  }
  if (generated.thinkingText) {
    result.thinking = { ...result.thinking, text: generated.thinkingText };
  }
  if (generated.statusEmojis) {
    result.statusEmojis = { ...result.statusEmojis, ...generated.statusEmojis };
  }
  return result;
}

/** A partial theme completed from the defaults, one section at a time. */
export function mergeTheme(partial: PartialGameTheme | undefined): GameTheme {
  const d = DEFAULT_GAME_THEME;
  if (!partial) return d;
  return {
    corners: { ...d.corners, ...partial.corners },
    background: { ...d.background, ...partial.background },
    player: { ...d.player, ...partial.player },
    gameMessage: {
      ...d.gameMessage,
      ...partial.gameMessage,
      textEffectScope: { ...d.gameMessage.textEffectScope, ...partial.gameMessage?.textEffectScope },
    },
    cards: { ...d.cards, ...partial.cards },
    thinking: { ...d.thinking, ...partial.thinking },
    typography: { ...d.typography, ...partial.typography },
    statusFields: { ...d.statusFields, ...partial.statusFields },
    header: { ...d.header, ...partial.header },
    divider: { ...d.divider, ...partial.divider },
    statusEmojis: { ...d.statusEmojis, ...partial.statusEmojis },
  } as GameTheme;
}

/** A field's emoji, matched exactly first and then ignoring case. */
export function statusEmoji(theme: GameTheme, field: string): string {
  const exact = theme.statusEmojis[field];
  if (exact) return exact;
  const lower = field.toLowerCase();
  for (const [key, emoji] of Object.entries(theme.statusEmojis)) {
    if (key.toLowerCase() === lower) return emoji;
  }
  return "";
}

const TINTS: Partial<Record<BackgroundTint, string>> = {
  warm: "rgba(251, 191, 36, 0.08)",
  cool: "rgba(34, 211, 238, 0.08)",
  dark: "#0f0f1a",
  black: "#000000",
  pink: "rgba(236, 72, 153, 0.12)",
  green: "rgba(34, 197, 94, 0.10)",
  blue: "rgba(59, 130, 246, 0.10)",
  violet: "rgba(139, 92, 246, 0.10)",
  darkCyan: "#0a1a1f",
  darkViolet: "#1a0f2e",
  darkBlue: "#0a0f1a",
  darkRose: "#1a0a0f",
};

/** Card colours dark enough that an error on them needs light red, not dark. */
const DARK_CARDS = new Set([
  "dark", "black", "blue", "green", "red", "amber", "violet", "rose", "cyan", "pink", "orange",
]);

/** Looks a name up in a colour table, falling back to a name known to be there. */
function pick<T>(table: Record<string, T>, name: string | undefined, fallback: string): T {
  return (name ? table[name] : undefined) ?? (table[fallback] as T);
}

/** The custom properties the stylesheet reads, keyed exactly as the platform names them. */
export function cssVars(theme: GameTheme): Record<string, string> {
  const corner = pick(THEME_COLORS, theme.corners.color, "amber");
  const player = pick(THEME_COLORS, theme.player.color, "cyan");
  const playerBorder = pick(THEME_COLORS, theme.player.borderColor, "cyan");
  const gameBorder = pick(THEME_COLORS, theme.gameMessage.borderColor, "amber");
  const dropCap = pick(THEME_COLORS, theme.gameMessage.dropCapColor, "amber");
  const statusAccent = pick(THEME_COLORS, theme.statusFields?.accentColor, "amber");
  const statusBorder = pick(THEME_COLORS, theme.statusFields?.borderColor, "amber");
  const headerAccent = pick(THEME_COLORS, theme.header?.accentColor, "amber");
  const divider = pick(THEME_COLORS, theme.divider?.color, "amber");
  const darkPlayer = DARK_CARDS.has(theme.player.bgColor);

  return {
    "--game-corner-color": corner.primary,
    "--game-corner-color-light": corner.light,
    "--game-corner-color-dark": corner.dark,

    "--game-player-color": player.primary,
    "--game-player-color-light": player.light,
    "--game-player-color-dark": player.dark,
    "--game-player-bg": pick(CARD_BG_COLORS, theme.player.bgColor, "white").solid,
    "--game-player-font-color": pick(FONT_COLORS, theme.player.fontColor, "dark"),
    "--game-player-border-color": playerBorder.primary,

    "--game-ai-bg": pick(CARD_BG_COLORS, theme.gameMessage.bgColor, "white").solid,
    "--game-ai-bg-alpha": pick(CARD_BG_COLORS, theme.gameMessage.bgColor, "white").alpha,
    "--game-ai-font-color": pick(FONT_COLORS, theme.gameMessage.fontColor, "dark"),
    "--game-ai-border-color": gameBorder.primary,
    "--game-drop-cap-color": dropCap.primary,

    "--game-border-width": pick(CARD_BORDER_THICKNESSES, theme.cards.borderThickness, "thin"),
    "--game-message-font": pick(THEME_FONTS, theme.typography.messages, "sans"),
    "--game-bg-tint": TINTS[theme.background.tint] ?? "transparent",

    "--game-image-highlight": gameBorder.primary,
    "--game-image-highlight-bg": gameBorder.bg,

    "--game-status-bg": pick(CARD_BG_COLORS, theme.statusFields?.bgColor, "creme").solid,
    "--game-status-accent": statusAccent.primary,
    "--game-status-border": statusBorder.primary,
    "--game-status-font": pick(FONT_COLORS, theme.statusFields?.fontColor, "dark"),

    "--game-header-bg": pick(CARD_BG_COLORS, theme.header?.bgColor, "white").solid,
    "--game-header-font": pick(FONT_COLORS, theme.header?.fontColor, "dark"),
    "--game-header-accent": headerAccent.primary,

    "--game-divider-color": divider.primary,

    "--game-error-bg": darkPlayer ? "rgba(251, 113, 133, 0.15)" : "rgba(239, 68, 68, 0.1)",
    "--game-error-border": darkPlayer ? "rgba(251, 113, 133, 0.4)" : "rgba(239, 68, 68, 0.4)",
    "--game-error-text": darkPlayer ? "#fb7185" : "#b91c1c",
  };
}
