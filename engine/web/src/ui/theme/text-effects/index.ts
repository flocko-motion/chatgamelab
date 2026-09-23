/**
 * The theme's text effects, ported from the platform's player
 * (game-player-v2/components/text-effects) so a message looks the same in both.
 * Classes and keyframes live in text-effects.css.
 */
import type { TextEffect } from "../types.js";
import { DecryptedText } from "./decrypted.js";
import { FadeInText } from "./fade-in.js";
import { FlickerText } from "./flicker.js";
import { GlitchText } from "./glitch.js";
import { RetroText, StyledText } from "./styled.js";
import { PlainText, reducedMotion } from "./text.js";

/** Paints text into one element with an effect, as the text grows. */
export interface TextRenderer {
  /** The element's full text so far. Called again each time it grows while streaming, and possibly with the same text. */
  update(text: string): void;
  /** Stops every timer; the element keeps showing the final text. */
  dispose(): void;
}

/** The effects that are a class on a span; reduced motion is handled in the CSS. */
const STYLED = {
  inkBleed: "fx-ink-bleed",
  parchmentBurn: "fx-parchment-burn",
  rainbow: "fx-rainbow",
  frost: "fx-frost",
  emberGlow: "fx-ember-glow",
  shadowPulse: "fx-shadow-pulse",
  bloodDrip: "fx-blood-drip",
} as const satisfies Partial<Record<TextEffect, string>>;

export function textRenderer(effect: TextEffect, element: HTMLElement): TextRenderer {
  switch (effect) {
    case "none":
      return new PlainText(element);
    case "retro":
      return new RetroText(element);
    case "decrypted":
      return reducedMotion() ? new PlainText(element) : new DecryptedText(element);
    case "glitch":
      return reducedMotion() ? new PlainText(element) : new GlitchText(element);
    case "flicker":
      return reducedMotion() ? new PlainText(element) : new FlickerText(element);
    case "fadeIn":
      return reducedMotion() ? new PlainText(element) : new FadeInText(element);
    default:
      return new StyledText(element, STYLED[effect]);
  }
}
