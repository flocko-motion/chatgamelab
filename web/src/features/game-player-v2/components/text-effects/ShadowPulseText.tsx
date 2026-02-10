/**
 * ShadowPulseText effect for AI messages.
 *
 * Text shadow grows and shrinks rhythmically, like light from a
 * swinging lamp in a dark room. Moody, cinematic feel.
 * Designed for noir theme.
 */

const KEYFRAMES = `
@keyframes shadowPulse {
  0%, 100% {
    text-shadow:
      2px 2px 4px rgba(0, 0, 0, 0.8),
      -1px -1px 3px rgba(0, 0, 0, 0.4);
    opacity: 0.92;
  }
  50% {
    text-shadow:
      4px 4px 12px rgba(0, 0, 0, 0.9),
      -2px -2px 8px rgba(0, 0, 0, 0.5),
      0 0 20px rgba(0, 0, 0, 0.3);
    opacity: 1;
  }
}
`;

let keyframesInjected = false;

function ensureKeyframes() {
  if (keyframesInjected) return;
  if (typeof document === 'undefined') return;
  const style = document.createElement('style');
  style.textContent = KEYFRAMES;
  document.head.appendChild(style);
  keyframesInjected = true;
}

interface ShadowPulseTextProps {
  text: string;
}

export function ShadowPulseText({ text }: ShadowPulseTextProps) {
  ensureKeyframes();

  return (
    <span
      style={{
        animation: 'shadowPulse 5s ease-in-out infinite',
      }}
    >
      {text}
    </span>
  );
}
