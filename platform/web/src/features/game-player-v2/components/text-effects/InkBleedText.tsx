/**
 * InkBleedText effect for AI messages.
 *
 * Text starts slightly blurry/spread and sharpens, like ink drying on paper.
 * Designed for medieval, mystery, pirate themes.
 */

const KEYFRAMES = `
@keyframes inkBleed {
  0% {
    filter: blur(2px);
    opacity: 0.6;
    letter-spacing: 0.05em;
  }
  100% {
    filter: blur(0);
    opacity: 1;
    letter-spacing: normal;
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

interface InkBleedTextProps {
  text: string;
}

export function InkBleedText({ text }: InkBleedTextProps) {
  ensureKeyframes();

  return (
    <span
      style={{
        animation: 'inkBleed 1.2s ease-out forwards',
      }}
    >
      {text}
    </span>
  );
}
