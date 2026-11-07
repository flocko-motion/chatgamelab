/**
 * RetroText effect for AI messages.
 *
 * Pure CSS CRT monitor effect with scanlines, phosphor glow, and slight flicker.
 * Designed for the terminal/retro theme.
 */

interface RetroTextProps {
  text: string;
}

const styles = {
  wrapper: {
    position: "relative" as const,
    display: "inline",
  },
  text: {
    textShadow: "0 0 4px currentColor, 0 0 8px currentColor",
    animation: "retroFlicker 4s infinite",
  },
  scanlines: {
    position: "absolute" as const,
    inset: "-4px -2px",
    background:
      "repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0, 0, 0, 0.15) 2px, rgba(0, 0, 0, 0.15) 4px)",
    pointerEvents: "none" as const,
    zIndex: 1,
    borderRadius: "2px",
  },
} as const;

// Inject keyframes once via a <style> tag (idempotent per component instance)
const KEYFRAMES = `
@keyframes retroFlicker {
  0%, 100% { opacity: 1; }
  92% { opacity: 1; }
  93% { opacity: 0.85; }
  94% { opacity: 1; }
  96% { opacity: 0.90; }
  97% { opacity: 1; }
}
`;

let keyframesInjected = false;

function ensureKeyframes() {
  if (keyframesInjected) return;
  if (typeof document === "undefined") return;
  const style = document.createElement("style");
  style.textContent = KEYFRAMES;
  document.head.appendChild(style);
  keyframesInjected = true;
}

export function RetroText({ text }: RetroTextProps) {
  ensureKeyframes();

  return (
    <span style={styles.wrapper}>
      <span style={styles.text}>{text}</span>
      <span style={styles.scanlines} aria-hidden="true" />
    </span>
  );
}
