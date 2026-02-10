/**
 * FrostText effect for AI messages.
 *
 * Continuous freeze/thaw breathing cycle — text slowly crystallizes with
 * an icy blue glow, holds, then thaws back to normal, and repeats.
 * Designed for snowy, fairy themes.
 */

const KEYFRAMES = `
@keyframes frostBreathe {
  0%, 100% {
    filter: blur(0);
    letter-spacing: 0;
    text-shadow:
      0 0 3px rgba(147, 197, 253, 0.3),
      0 0 1px rgba(255, 255, 255, 0.3);
  }
  40% {
    filter: blur(0.5px);
    letter-spacing: 0.1px;
    text-shadow:
      0 0 12px rgba(96, 165, 250, 0.8),
      0 0 24px rgba(147, 197, 253, 0.5),
      0 0 40px rgba(186, 230, 253, 0.3),
      0 0 4px rgba(255, 255, 255, 0.7);
  }
  60% {
    filter: blur(0.5px);
    letter-spacing: 0.1px;
    text-shadow:
      0 0 12px rgba(96, 165, 250, 0.8),
      0 0 24px rgba(147, 197, 253, 0.5),
      0 0 40px rgba(186, 230, 253, 0.3),
      0 0 4px rgba(255, 255, 255, 0.7);
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

interface FrostTextProps {
  text: string;
}

export function FrostText({ text }: FrostTextProps) {
  ensureKeyframes();

  return (
    <span
      style={{
        animation: "frostBreathe 8s ease-in-out infinite",
      }}
    >
      {text}
    </span>
  );
}
