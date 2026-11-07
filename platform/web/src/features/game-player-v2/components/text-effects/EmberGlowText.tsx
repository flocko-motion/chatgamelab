/**
 * EmberGlowText effect for AI messages.
 *
 * Warm pulsing glow on text edges, like embers in a fire.
 * Designed for fire, desert, western themes.
 */

const KEYFRAMES = `
@keyframes emberPulse {
  0%, 100% {
    text-shadow:
      0 0 4px rgba(251, 146, 60, 0.4),
      0 0 8px rgba(239, 68, 68, 0.2);
  }
  50% {
    text-shadow:
      0 0 8px rgba(251, 146, 60, 0.6),
      0 0 16px rgba(239, 68, 68, 0.3),
      0 0 24px rgba(234, 179, 8, 0.15);
  }
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

interface EmberGlowTextProps {
  text: string;
}

export function EmberGlowText({ text }: EmberGlowTextProps) {
  ensureKeyframes();

  return (
    <span
      style={{
        animation: "emberPulse 10s ease-in-out infinite",
        display: "inline",
        padding: "0 2px",
      }}
    >
      {text}
    </span>
  );
}
