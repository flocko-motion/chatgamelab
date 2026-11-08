/**
 * Puts a theme on the game half: its colours as custom properties, and its
 * shape choices as attributes and strings the stylesheet reads. Nothing is
 * rebuilt, so a theme can change under a running conversation.
 */
import { cssVars } from "./resolve.js";
import type { DividerStyle, GameTheme, PlayerIndicator, StreamingCursor } from "./types.js";

/**
 * The platform's player drew diamond, arrow and star as ">", its fallback;
 * thirteen presets ask for them, so they get the symbols their names describe.
 */
const INDICATORS: Record<PlayerIndicator, string> = {
  dot: "•",
  chevron: ">",
  pipe: "|",
  cursor: "▌",
  underscore: "_",
  diamond: "◆",
  arrow: "➤",
  star: "★",
  none: "",
};

const DIVIDERS: Record<DividerStyle, string> = {
  dot: "•",
  line: "-",
  dots: "• • •",
  diamond: "◆",
  star: "✦",
  dash: "---",
  none: "",
};

const CURSORS: Record<StreamingCursor, string> = {
  dots: "•••",
  block: "█",
  pipe: "|",
  underscore: "_",
  none: "",
};

/**
 * Sets `theme` on `root`, replacing whatever theme was there. `preset` names
 * the preset it came from, which a hand-drawn skin keys on (theme/skins).
 */
export function applyTheme(root: HTMLElement, theme: GameTheme, preset: string): void {
  root.dataset["preset"] = preset;
  for (const [name, value] of Object.entries(cssVars(theme))) {
    root.style.setProperty(name, value);
  }
  root.style.setProperty("--game-player-indicator", quote(INDICATORS[theme.player.indicator] ?? ">"));
  root.style.setProperty("--game-divider-symbol", quote(DIVIDERS[theme.divider.style] ?? "•"));
  const cursor = theme.thinking.streamingCursor ?? "dots";
  root.style.setProperty("--game-streaming-cursor", quote(CURSORS[cursor] ?? "|"));

  const positions = theme.corners.positions ?? { topLeft: true, topRight: false, bottomLeft: false, bottomRight: true };
  const data = root.dataset;
  data["corners"] = theme.corners.style;
  flag(root, "cornerTl", positions.topLeft);
  flag(root, "cornerTr", positions.topRight);
  flag(root, "cornerBl", positions.bottomLeft);
  flag(root, "cornerBr", positions.bottomRight);
  flag(root, "cornerBlink", theme.corners.blink ?? false);
  flag(root, "dropCap", theme.gameMessage.dropCap);
  data["divider"] = theme.divider.style;
  data["indicator"] = theme.player.indicator;
  flag(root, "indicatorBlink", theme.player.indicatorBlink);
  data["cursor"] = cursor;
}

/** A string as a CSS string literal, for `content: var(...)`. */
function quote(text: string): string {
  return JSON.stringify(text);
}

function flag(root: HTMLElement, name: string, on: boolean): void {
  if (on) root.dataset[name] = "";
  else delete root.dataset[name];
}
