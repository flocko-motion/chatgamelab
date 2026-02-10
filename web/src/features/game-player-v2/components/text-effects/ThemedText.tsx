/**
 * ThemedText - Applies the active text effect from the game theme.
 *
 * Shared helper so any component can render text with the current
 * theme's text effect (retro, glitch, decrypted, etc.) without
 * duplicating the effect-selection logic.
 *
 * The `scope` prop controls which area is requesting the effect.
 * The theme's `textEffectScope` config determines whether the effect
 * is actually applied for that area.
 */

import { useGamePlayerContext } from "../../context";
import { useGameTheme } from "../../theme";
import { DecryptedText } from "./DecryptedText";
import { GlitchText } from "./GlitchText";
import { RetroText } from "./RetroText";
import { HandwrittenText } from "./HandwrittenText";
import { InkBleedText } from "./InkBleedText";
import { FadeInText } from "./FadeInText";
import { ParchmentBurnText } from "./ParchmentBurnText";
import { FlickerText } from "./FlickerText";
import { RainbowText } from "./RainbowText";
import { FrostText } from "./FrostText";
import { EmberGlowText } from "./EmberGlowText";
import { ShadowPulseText } from "./ShadowPulseText";
import { BloodDripText } from "./BloodDripText";

export type TextEffectScopeKey =
  | "gameMessages"
  | "playerMessages"
  | "statusFields";

interface ThemedTextProps {
  text: string;
  /** Which area is rendering this text (defaults to 'gameMessages') */
  scope?: TextEffectScopeKey;
}

export function ThemedText({ text, scope = "gameMessages" }: ThemedTextProps) {
  const { textEffectsEnabled } = useGamePlayerContext();
  const { theme } = useGameTheme();
  const textEffect = theme.gameMessage?.textEffect ?? "none";
  const effectScope = theme.gameMessage?.textEffectScope;

  // Check if effects are globally disabled or disabled for this scope
  const isScopeEnabled = effectScope ? effectScope[scope] : true;

  if (textEffect === "none" || !textEffectsEnabled || !isScopeEnabled) {
    return <>{text}</>;
  }

  switch (textEffect) {
    case "decrypted":
      return <DecryptedText text={text} />;
    case "glitch":
      return <GlitchText text={text} />;
    case "retro":
      return <RetroText text={text} />;
    case "handwritten":
      return <HandwrittenText text={text} />;
    case "inkBleed":
      return <InkBleedText text={text} />;
    case "fadeIn":
      return <FadeInText text={text} />;
    case "parchmentBurn":
      return <ParchmentBurnText text={text} />;
    case "flicker":
      return <FlickerText text={text} />;
    case "rainbow":
      return <RainbowText text={text} />;
    case "frost":
      return <FrostText text={text} />;
    case "emberGlow":
      return <EmberGlowText text={text} />;
    case "shadowPulse":
      return <ShadowPulseText text={text} />;
    case "bloodDrip":
      return <BloodDripText text={text} />;
    default:
      return <>{text}</>;
  }
}
